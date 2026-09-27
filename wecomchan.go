package main
import (
 "bytes"; "context"; "crypto/subtle"; "encoding/json"; "errors"; "fmt"; "io"; "log"; "net/http"; "net/url"; "os"; "strings"; "sync"; "time"
)
const listenAddr="127.0.0.1:8080"
const maxBody=64*1024
type Config struct{SendKey,WecomCID,Secret,AgentID,ToUser string}
type tokenCache struct{mu sync.Mutex; token string; expiresAt time.Time}
var cache tokenCache
func env(k,f string)string{if v:=os.Getenv(k);v!=""{return v};return f}
func loadConfig()Config{return Config{env("SENDKEY",""),env("WECOM_CID",""),env("WECOM_SECRET",""),env("WECOM_AID",""),env("WECOM_TOUID","@all")}}
func validateConfig(c Config)error{for n,v:=range map[string]string{"SENDKEY":c.SendKey,"WECOM_CID":c.WecomCID,"WECOM_SECRET":c.Secret,"WECOM_AID":c.AgentID}{if v==""{return fmt.Errorf("missing required environment variable %s",n)}};return nil}
func jsonResponse(w http.ResponseWriter,s int,v any){w.Header().Set("Content-Type","application/json; charset=utf-8");w.WriteHeader(s);_=json.NewEncoder(w).Encode(v)}
func getAccessToken(ctx context.Context,c Config)(string,error){
 cache.mu.Lock();if cache.token!=""&&time.Until(cache.expiresAt)>2*time.Minute{t:=cache.token;cache.mu.Unlock();return t,nil};cache.mu.Unlock()
 cache.mu.Lock();defer cache.mu.Unlock();if cache.token!=""&&time.Until(cache.expiresAt)>2*time.Minute{return cache.token,nil}
 u:="https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid="+url.QueryEscape(c.WecomCID)+"&corpsecret="+url.QueryEscape(c.Secret)
 req,e:=http.NewRequestWithContext(ctx,http.MethodGet,u,nil);if e!=nil{return "",e};resp,e:=http.DefaultClient.Do(req);if e!=nil{return "",fmt.Errorf("get access_token request failed: %w",e)};defer resp.Body.Close()
 var b struct{ErrCode int `json:"errcode"`;ErrMsg string `json:"errmsg"`;Token string `json:"access_token"`;Expires int `json:"expires_in"`}
 if e=json.NewDecoder(resp.Body).Decode(&b);e!=nil{return "",fmt.Errorf("decode access_token response: %w",e)};if b.ErrCode!=0||b.Token==""{return "",fmt.Errorf("wecom gettoken failed: errcode=%d errmsg=%s",b.ErrCode,b.ErrMsg)}
 ttl:=time.Duration(b.Expires)*time.Second;if ttl<=0{ttl=2*time.Hour};cache.token=b.Token;cache.expiresAt=time.Now().Add(ttl);return b.Token,nil
}
func sendMessage(ctx context.Context,token string,c Config,mt,to,msg string)(map[string]any,error){
 if to==""{to=c.ToUser};p:=map[string]any{"touser":to,"agentid":c.AgentID,"msgtype":mt,"duplicate_check_interval":600}
 switch mt{case "text":p["text"]=map[string]string{"content":msg};case "markdown":p["markdown"]=map[string]string{"content":msg};default:return nil,fmt.Errorf("unsupported msg_type %q; use text or markdown",mt)}
 b,e:=json.Marshal(p);if e!=nil{return nil,e};u:="https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token="+url.QueryEscape(token);req,e:=http.NewRequestWithContext(ctx,http.MethodPost,u,bytes.NewReader(b));if e!=nil{return nil,e};req.Header.Set("Content-Type","application/json")
 resp,e:=http.DefaultClient.Do(req);if e!=nil{return nil,fmt.Errorf("send message request failed: %w",e)};defer resp.Body.Close();bb,e:=io.ReadAll(io.LimitReader(resp.Body,maxBody));if e!=nil{return nil,e};var r map[string]any;if e=json.Unmarshal(bb,&r);e!=nil{return nil,e};if code,ok:=r["errcode"].(float64);ok&&code!=0{return r,errors.New("wecom message/send returned an error")};return r,nil
}
type requestPayload struct{SendKey string `json:"sendkey"`;MsgType string `json:"msg_type"`;Msg string `json:"msg"`;ToUser string `json:"to_user"`}
func handler(c Config)http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){
 if r.Method!=http.MethodGet&&r.Method!=http.MethodPost{w.Header().Set("Allow","GET, POST");jsonResponse(w,405,map[string]any{"ok":false,"error":"method not allowed"});return}
 var p requestPayload
 if r.Method==http.MethodGet{q:=r.URL.Query();p.SendKey=q.Get("sendkey");p.MsgType=q.Get("msg_type");p.Msg=q.Get("msg");p.ToUser=q.Get("to_user")}else{r.Body=http.MaxBytesReader(w,r.Body,maxBody);if e:=json.NewDecoder(r.Body).Decode(&p);e!=nil{jsonResponse(w,400,map[string]any{"ok":false,"error":"invalid JSON body"});return}}
 k:=p.SendKey;if k==""{a:=strings.TrimSpace(r.Header.Get("Authorization"));if strings.HasPrefix(a,"Bearer "){k=strings.TrimSpace(strings.TrimPrefix(a,"Bearer "))}};if k==""{k=strings.TrimSpace(r.Header.Get("X-API-Key"))}
 if subtle.ConstantTimeCompare([]byte(k),[]byte(c.SendKey))!=1{jsonResponse(w,401,map[string]any{"ok":false,"error":"invalid credentials"});return};if p.Msg==""{jsonResponse(w,400,map[string]any{"ok":false,"error":"msg is required"});return};if p.MsgType==""{p.MsgType="text"}
 ctx,cancel:=context.WithTimeout(r.Context(),10*time.Second);defer cancel();token,e:=getAccessToken(ctx,c);if e!=nil{log.Printf("get access token failed: %v",e);jsonResponse(w,502,map[string]any{"ok":false,"error":"upstream token request failed"});return}
 result,e:=sendMessage(ctx,token,c,p.MsgType,p.ToUser,p.Msg);if e!=nil&&result!=nil{if code,ok:=result["errcode"].(float64);ok&&int(code)==42001{cache.mu.Lock();cache.token="";cache.expiresAt=time.Time{};cache.mu.Unlock();if t2,te:=getAccessToken(ctx,c);te==nil{result,e=sendMessage(ctx,t2,c,p.MsgType,p.ToUser,p.Msg)}}}
 if e!=nil{log.Printf("send message failed: %v",e);if result!=nil{jsonResponse(w,502,result)}else{jsonResponse(w,502,map[string]any{"ok":false,"error":"upstream message request failed"})};return};jsonResponse(w,200,result)
}}
func main(){log.SetFlags(log.LstdFlags|log.Lshortfile);c:=loadConfig();if e:=validateConfig(c);e!=nil{log.Fatal(e)};m:=http.NewServeMux();m.HandleFunc("/health",func(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodGet{w.WriteHeader(405);return};jsonResponse(w,200,map[string]any{"ok":true})});m.HandleFunc("/wecomchan",handler(c));s:=&http.Server{Addr:listenAddr,Handler:m,ReadHeaderTimeout:5*time.Second,ReadTimeout:15*time.Second,WriteTimeout:15*time.Second,IdleTimeout:60*time.Second};log.Printf("wecomx listening on %s",listenAddr);log.Fatal(s.ListenAndServe())}

# go-rsync

Go-rsync is a golang rsync wrapper



#### Usage:

```
func main() {

   ssh := rsync.SSH{Host: "172.24.120.46", Port: 22, User: "root", Password: "password"}
   target := "/root"
	
   // check can rsync
   err := rsync.CheckRsync(ssh, target)
   if err != nil {
      fmt.Println("rsync err: %v", err)
      return
   }

   r, err := rsync.NewRsync(nil, ssh)
   if err != nil {
      fmt.Println("newrsync err: %v", err)
      return
   }

   err = r.Copy(target, "./README.md")
   if err != nil {
      fmt.Println("rsync err: %v", err)
      return
   }
}
```
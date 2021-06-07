# Cacher

Common cacher library for some golang projects


# Usage example

This library can be used for store information in cache files. That information will be get from the server and the stored in a file.

```golang
import (
    "net/http"
    
    "https://github.com/relex/gotils/cacher"
)

req, _ := http.NewRequest("GET", "http://localhost:8080", nil)
req.Header.Add("PRIVATE-TOKEN", "myToken")
// body contains the current information which was stored in the file
body, err := cacher.GetFromURLOrDefaultCache(req, "myCacheFolder")
if err != nil {
    //YOUR STUFF
}
...
```

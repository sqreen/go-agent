package api

// Temporary hack to for `go get` to include gogoproto subdirectory. The
// `*.proto` files include it and it is thus required to compile them. Since we
// download it using `go get`, we thus need to force this dependency. Maybe `go
// get` will provide to force this in the future.
import _ "github.com/gogo/protobuf/gogoproto"

// OAuth 2.0 server library for the Go programming language
//
//     package main
//     import (
//         "net/http"
//         "gopkg.in/oauth2.v3/manage"
//         "gopkg.in/oauth2.v3/server"
//         "gopkg.in/oauth2.v3/store"
//     )
//     func main() {
//         manager := manage.NewDefaultManager()
//         manager.MustTokenStorage(store.NewMemoryTokenStore())
//         manager.MapClientStorage(store.NewTestClientStore())
//         srv := server.NewDefaultServer(manager)
//         http.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
//             srv.HandleAuthorizeRequest(w, r)
//         })
//         http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
//             srv.HandleTokenRequest(w, r)
//         })
//         http.ListenAndServe(":9096", nil)
//     }

package oauth2

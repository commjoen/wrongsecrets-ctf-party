package routes

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/juice-shop/multi-juicer/balancer/pkg/bundle"
	"github.com/juice-shop/multi-juicer/balancer/pkg/signutil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var instanceUpCache = map[string]int64{}

func clearInstanceUpCache() {
	instanceUpCache = map[string]int64{}
}

// newReverseProxy creates a reverse proxy for a given target URL.
func newReverseProxy(target string) *httputil.ReverseProxy {
	url, err := url.Parse(target)
	if err != nil {
		log.Fatalf("Failed to parse target URL: %v", err)
	}
	return httputil.NewSingleHostReverseProxy(url)
}

// HandleProxy determines the JuiceShop instance of the Team based on the "balancer" cookie and proxies the request to the corresponding JuiceShop instance.
func handleProxy(bundle *bundle.Bundle) http.Handler {
	return http.HandlerFunc(
		func(responseWriter http.ResponseWriter, req *http.Request) {
			teamSigned, err := req.Cookie("balancer")
			if err != nil {
				http.SetCookie(responseWriter, &http.Cookie{Name: "balancer", Path: "/", MaxAge: -1})
				http.Redirect(responseWriter, req, "/balancer", http.StatusFound)
				return
			}
			team, err := signutil.Unsign(teamSigned.Value, bundle.Config.CookieConfig.SigningKey)
			if err != nil {
				bundle.Log.Printf("Invalid cookie signature, unsetting cookie and redirecting to balancer page.")
				http.SetCookie(responseWriter, &http.Cookie{Name: "balancer", Path: "/", MaxAge: -1})
				http.Redirect(responseWriter, req, "/balancer", http.StatusFound)
				return
			}
			if team == "" {
				bundle.Log.Printf("Empty team in signed cookie! Unsetting cookie and redirecting to balancer page.")
				http.SetCookie(responseWriter, &http.Cookie{Name: "balancer", Path: "/", MaxAge: -1})
				http.Redirect(responseWriter, req, "/balancer", http.StatusFound)
				return
			}

			if !wasInstanceUptimeStatusCheckedRecently(team) {
				if isInstanceUp(bundle, team) {
					instanceUpCache[team] = time.Now().UnixMilli()
				} else {
					bundle.Log.Printf("Instance for team (%s) is down. Redirecting to balancer page.", team)
					http.Redirect(responseWriter, req, fmt.Sprintf("/balancer/?msg=instance-restarting&teamname=%s", team), http.StatusFound)
					return
				}
			}

			target := bundle.GetJuiceShopUrlForTeam(team, bundle)
			bundle.Log.Printf("Proxying request for team (%s): %s %s to %s", team, req.Method, req.URL, target)
			// Rewrite the request to the target server
			newReverseProxy(target).ServeHTTP(responseWriter, req)
		})
}

// checks if the instance uptime status was checked in the last ten seconds by looking into the instanceUpCache
func wasInstanceUptimeStatusCheckedRecently(team string) bool {
	lastConnect, ok := instanceUpCache[team]
	return ok && lastConnect > time.Now().Add(-10*time.Second).UnixMilli()
}

func isInstanceUp(bundle *bundle.Bundle, team string) bool {
	deployment, err := bundle.ClientSet.AppsV1().Deployments(bundle.RuntimeEnvironment.Namespace).Get(context.Background(), fmt.Sprintf("juiceshop-%s", team), metav1.GetOptions{})

	if errors.IsNotFound(err) {
		return false
	} else if err != nil {
		bundle.Log.Printf("Failed to lookup if a instance is up in the kubernetes api. Assuming it's missing: %s", err)
		return false
	} else {
		return deployment.Status.ReadyReplicas > 0
	}
}

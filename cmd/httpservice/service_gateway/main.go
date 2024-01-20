package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

type Route struct {
	Name    string `mapstructure:"name"`
	Context string `mapstructure:"context"`
	Target  string `mapstructure:"target"`
}

type GatewayConfig struct {
	ListenAddr string  `mapstructure:"listenAddr"`
	Routes     []Route `mapstructure:"routes"`
}

func main() {
	viper.AddConfigPath("./")
	viper.SetConfigType("yaml")

	viper.SetConfigName("default")

	err := viper.ReadInConfig()
	if err != nil {
		log.Println("WARNING: could not load configuration", err)
	}

	viper.AutomaticEnv()

	gatewayConfig := &GatewayConfig{}

	viper.UnmarshalKey("gateway", gatewayConfig)
	if err != nil {
		log.Println("ERROR: cannot unmarshal key", err)
		panic(err)
	}

	log.Println("Initializing routes...")

	r := mux.NewRouter()

	for _, route := range gatewayConfig.Routes {
		// Returns a proxy for the target url.
		proxy, err := NewProxy(route.Target)
		if err != nil {
			panic(err)
		}

		log.Printf("Mapping '%v' | %v ---> %v", route.Name, route.Context, route.Target)

		r.HandleFunc(route.Context+"/{targetPath:.*}", NewHandler(proxy))
	}

	log.Printf("Started server on %v", gatewayConfig.ListenAddr)
	log.Fatal(http.ListenAndServe(gatewayConfig.ListenAddr, r))
}

func NewProxy(targetUrl string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(targetUrl)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = func(response *http.Response) error {
		dumpedResponse, err := httputil.DumpResponse(response, false)
		if err != nil {
			return err
		}

		log.Println("RESPONSE: \r\n", string(dumpedResponse))
		return nil
	}

	return proxy, nil
}

func NewHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = mux.Vars(r)["targetPath"]
		log.Println("REQUEST URL: ", r.URL.String())

		proxy.ServeHTTP(w, r)
	}
}

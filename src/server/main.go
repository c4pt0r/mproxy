package main

import (
    "proxy"
)

func main() {
    cfg, _ := proxy.NewConfig()
    srv := proxy.NewServer(cfg)
    srv.Start()
}

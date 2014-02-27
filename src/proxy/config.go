package proxy

type config struct {
    addr string
    dbAddr string
    dbName string
}

func NewConfig() *config {
    return &config{
        "localhost:3307",
        "localhost:3306",
        "mysql",
    }
}

package proxy

type config struct {
	addr       string
	dbAddr     string
	dbName     string
	dbUser     string
	dbPassword string
}

func NewConfig() *config {
	return &config{
		addr:       "localhost:3307",
		dbAddr:     "localhost:3306",
		dbName:     "mysql",
		dbUser:     "root",
		dbPassword: "root",
	}
}

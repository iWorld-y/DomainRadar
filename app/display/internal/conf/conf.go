package conf

type Bootstrap struct {
	Server *Server
	Data   *Data
}

type Server struct {
	Http *HTTP
}

type HTTP struct {
	Addr    string
	Timeout string
}

type Data struct {
	Database *Database
}

type Database struct {
	Driver string
	Source string
}

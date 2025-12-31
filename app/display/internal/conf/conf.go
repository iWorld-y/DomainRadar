package conf

type Bootstrap struct {
	Server *Server
	Data   *Data
	Auth   *Auth
	Radar  *Radar
}

type Auth struct {
	JwtKey string
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

type Radar struct {
	Llm         *LLM         `json:"llm"`
	Search      *Search      `json:"search"`
	UserPersona string       `json:"user_persona"`
	Domains     []string     `json:"domains"`
	Log         *Log         `json:"log"`
	Concurrency *Concurrency `json:"concurrency"`
	Db          *DB          `json:"db"`
}

type LLM struct {
	BaseUrl string `json:"base_url"`
	ApiKey  string `json:"api_key"`
	Model   string `json:"model"`
}

type Search struct {
	Provider string   `json:"provider"`
	Tavily   *Tavily  `json:"tavily"`
	Searxng  *SearXNG `json:"searxng"`
}

type Tavily struct {
	ApiKey string `json:"api_key"`
}

type SearXNG struct {
	BaseUrl string `json:"base_url"`
	Timeout int32  `json:"timeout"`
}

type Log struct {
	Level string `json:"level"`
	File  string `json:"file"`
}

type Concurrency struct {
	Qps int32 `json:"qps"`
	Rpm int32 `json:"rpm"`
}

type DB struct {
	Host     string `json:"host"`
	Port     int32  `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

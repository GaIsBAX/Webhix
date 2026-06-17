package notify

type Options struct {
	Server    string
	AuthToken string
}

func DefaultOptions() Options {
	return Options{
		Server: "http://localhost:8080",
	}
}

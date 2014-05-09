package ego

type expr interface{}

type keyword struct {
	receiver  expr
	keywords  []string
	arguments []expr
	delegate  string
}

type binary struct {
	receiver expr
	operator string
	argument expr
	delegate string
}

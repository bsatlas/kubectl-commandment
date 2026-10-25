package root

// Config owns kubectl values that are semantically rooted above the selected command.
type Config struct {
	Kubeconfig string
	Context    string
	Namespace  string
}

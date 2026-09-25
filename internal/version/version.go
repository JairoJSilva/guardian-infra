package version

import "fmt"

var (
	// Version é a versão semântica oficial da plataforma Guardian SRE
	// Pode ser sobrescrita em tempo de compilação via:
	// -ldflags "-X 'guardian/internal/version.Version=x.y.z'"
	Version = "3.1.0"

	// GitCommit é o hash curto do commit do Git
	GitCommit = "dev"

	// BuildDate é a data e hora UTC da compilação
	BuildDate = "unknown"
)

// FullVersion retorna a versão formatada com commit e data de compilação
func FullVersion() string {
	if GitCommit != "dev" && GitCommit != "" && BuildDate != "unknown" && BuildDate != "" {
		return fmt.Sprintf("v%s (%s, %s)", Version, GitCommit, BuildDate)
	}
	return fmt.Sprintf("v%s", Version)
}

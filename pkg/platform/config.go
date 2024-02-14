package konstellation

type PlatformConfig struct {
	Mappings map[string]Mapping `yaml:"mappings"`
	Order    struct {
		Last []string `yaml:"last"`
	} `yaml:"order"`
	Relationships []Relationship `yaml:"relationships"`
	Queries       []Query        `yaml:"queries"`
}

type Mapping struct {
	Template   string `yaml:"template"`
	JsonPath   string `yaml:"jsonPath"`
	LabelField string `yaml:"labelField"`
	Label      string `yaml:"label"`
	NameField  string `yaml:"nameField"`
}

type Relationship struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Query struct {
	Name     string `yaml:"name"`
	Query    string `yaml:"query"`
	Template string `yaml:"template"`
}

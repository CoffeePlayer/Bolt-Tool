package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type BoltResult struct {
	Source      string
	Title       string
	Body        string
	URL         string
	CodeSnippet string
	Score       int
}

type BrainResult struct {
	EnglishQ   string         `json:"english_q"`
	OptimizedQ string         `json:"optimized_q"`
	Primary    string         `json:"primary_tokens"`
	Action     string         `json:"action_tokens"`
	Context    string         `json:"context_tokens"`
	Exclude    string         `json:"exclude_tokens"`
	Targets    map[string]int `json:"targets"`
	Insight    string         `json:"insight"`
}

var printMu sync.Mutex

var massiveTechDB = make(map[string]bool)
var techDictionary []string
var synonymDB map[string][]string
var routingDB map[string][]string

var (
	reCodeBlock   = regexp.MustCompile("(?s)```(.*?)(```|$)")
	reHTML        = regexp.MustCompile(`<[^>]*>`)
	reMDLinks     = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	reMDHeaders   = regexp.MustCompile(`(?m)^#+\s*`)
	reMDBold      = regexp.MustCompile(`\*\*|__`)
	reNL          = regexp.MustCompile(`\n{2,}`)
	rePunct       = regexp.MustCompile(`[^\w\s]`)
	reCodeExtract = regexp.MustCompile("(?s)```(.*?)\n(.*?)```")
)

var (
	ansiReset    = "\033[0m"
	ansiBold     = "\033[1m"
	ansiCyan     = "\033[36m"
	ansiGreen    = "\033[32m"
	ansiGray     = "\033[90m"
	ansiMagenta  = "\033[35m"
	ansiYellow   = "\033[33m"
	ansiBlueLine = "\033[44;37m"
)

var httpClient = &http.Client{
	Timeout: 12 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     60 * time.Second,
	},
}

func createIfMissing(filename string, defaultData string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		_ = os.WriteFile(filename, []byte(defaultData), 0644)
	}
}

func initDatabases() {
	defaultTechDB := `[
		"python", "javascript", "js", "java", "c++", "cpp", "c#", "csharp", "golang", "go", "rust", "php", "ruby", "swift", "kotlin", "dart",
		"typescript", "ts", "scala", "r", "perl", "lua", "bash", "shell", "html", "css", "sass", "less", "xml", "json", "yaml", "markdown",
		"react", "angular", "vue", "svelte", "next", "nuxt", "gatsby", "node", "nodejs", "express", "nest", "django", "flask", "fastapi",
		"spring", "laravel", "symfony", "rails", "asp", "dotnet", "sql", "mysql", "postgres", "postgresql", "oracle", "mssql", "nosql",
		"mongodb", "redis", "cassandra", "firebase", "supabase", "docker", "kubernetes", "k8s", "aws", "azure", "gcp", "terraform",
		"linux", "ubuntu", "debian", "centos", "mac", "macos", "windows", "git", "github", "gitlab", "bitbucket", "svn", "ci", "cd",
		"api", "rest", "graphql", "grpc", "soap", "http", "https", "tcp", "bug", "error", "exception", "crash", "leak", "memory", "cpu",
		"loop", "array", "list", "dict", "object", "class", "function", "variable", "const", "let", "var", "pointer", "string", "int",
		"bool", "boolean", "float", "double", "char", "byte", "struct", "interface", "algorithm", "sort", "search", "tree", "graph", "queue",
		"stack", "hash", "map", "set", "regex", "socket", "thread", "async", "await", "promise", "callback", "frontend", "backend",
		"fullstack", "devops", "ui", "ux", "server", "client", "browser", "app", "application", "software", "program", "script", "code",
		"ai", "ml", "machine learning", "llm", "blockchain", "web3", "security", "testing", "jest", "cypress", "selenium", "pytest", "tailwind",
		"pandas", "numpy", "tensorflow", "pytorch", "keras", "scikit", "matplotlib", "seaborn", "opencv", "nltk", "spacy", "hadoop", "spark",
		"kafka", "rabbitmq", "celery", "airflow", "ansible", "puppet", "chef", "vagrant", "nginx", "apache", "caddy", "haproxy", "traefik",
		"prometheus", "grafana", "elk", "logstash", "kibana", "datadog", "newrelic", "splunk", "auth0", "jwt", "oauth", "saml", "openid",
		"webpack", "babel", "vite", "rollup", "parcel", "eslint", "prettier", "husky", "npm", "yarn", "pnpm", "pip", "conda", "maven", "gradle",
		"cargo", "nuget", "gem", "composer", "solidity", "ethers", "web3js", "hardhat", "truffle", "ipfs", "metamask", "smart contract", "crypto",
		"ec2", "s3", "lambda", "fargate", "ecs", "eks", "rds", "dynamodb", "unity", "godot", "unreal", "xamarin", "ionic", "cordova", "electron",
		"tauri", "htmx", "alpinejs", "pinia", "vuex", "redux", "mobx", "zustand", "trpc", "webrtc", "websocket", "socketio", "pwa", "seo", "a11y",
		"i18n", "l10n", "csrf", "xss", "cors", "ddos", "owasp", "nmap", "wireshark", "kali", "powershell", "cmd", "zsh", "fish", "tmux", "vim",
		"neovim", "emacs", "vscode", "intellij", "eclipse", "xcode", "android studio", "x86", "arm", "riscv", "microcontroller", "arduino", "raspberry",
		"sveltekit", "solidjs", "qwik", "astroweb", "bootstrap", "materialui", "chakra", "figma", "jira", "confluence", "gitops", "argo", "flux",
		"helm", "istio", "linkerd", "consul", "vault", "boundary", "nomad", "packer", "pulumi", "cloudformation", "cdk", "serverless", "sst",
		"vercel", "netlify", "heroku", "render", "flyio", "digitalocean", "linode", "vultr", "appwrite", "nhost", "amplify", "clerk", "okta",
		"zig", "nim", "crystal", "elixir", "erlang", "clojure", "fsharp", "haskell", "ocaml", "prolog", "fortran", "cobol", "ada", "lisp", "groovy",
		"actix", "rocket", "axum", "gin", "echo", "fiber", "beego", "starlette", "tornado", "sanic", "falcon", "quart", "pyramid", "bottle",
		"huggingface", "transformers", "diffusers", "langchain", "llamaindex", "ollama", "vllm", "tensorrt", "cudnn", "cuda", "rocm", "opencl",
		"sqlite", "mariadb", "cockroachdb", "tidb", "neo4j", "arango", "influxdb", "timescaledb", "clickhouse", "snowflake", "bigquery", "redshift",
		"rabbitmq", "activemq", "zeromq", "nats", "pulsar", "sqs", "sns", "eventbridge", "kinesis", "pubsub", "grpc", "protobuf", "thrift", "avro",
		"sonarqube", "checkmarx", "fortify", "veracode", "snyk", "dependabot", "trivy", "grype", "syft", "terrascan", "checkov", "tfsec", "terragrunt",
		"istio", "envoy", "traefik", "kong", "tyk", "apigee", "mulesoft", "wcf", "wpf", "winforms", "maui", "blazor", "razor", "signalr", "entityframework",
		"drizzle", "prisma", "sequelize", "typeorm", "knex", "mongoose", "sqlalchemy", "gorm", "diesel", "sea-orm", "ent",
		"bun", "deno", "cloudflare workers", "wasm", "webassembly", "wasi", "emscripten",
		"playwright", "puppeteer", "vitest", "mocha", "chai", "jasmine", "karma", "storybook",
		"turbopack", "turborepo", "nx", "lerna", "changesets", "semantic-release",
		"supabase", "planetscale", "neon", "upstash", "convex", "fauna",
		"opentelemetry", "otel", "jaeger", "zipkin", "sentry", "bugsnag",
		"remix", "fresh", "hono", "elysia", "baojs", "nitro", "vinxi",
		"shadcn", "radix", "headlessui", "ark", "mantine", "antd", "daisyui", "flowbite"
	]`

	defaultSpellcheckDB := `[
		"python", "javascript", "golang", "react", "angular", "node", "error", "exception", "memory", "leak", "pointer", "database", "server", "linux",
		"docker", "kubernetes", "django", "flask", "spring", "laravel", "mysql", "typescript", "tailwind", "postgres", "graphql", "rust", "swift",
		"tensorflow", "pytorch", "elasticsearch", "microservices", "architecture", "authentication", "authorization", "middleware", "asynchronous",
		"concurrency", "parallelism", "deployment", "repository", "framework", "library", "dependency", "environment", "variable", "function",
		"component", "interface", "implementation", "algorithm", "optimization", "performance", "scalability", "reliability", "availability",
		"vulnerability", "encryption", "decryption", "cryptography", "blockchain", "artificial", "intelligence", "machine", "learning", "dataset",
		"frontend", "backend", "fullstack", "developer", "engineer", "programmer", "software", "application", "system", "network", "protocol",
		"multithreading", "synchronous", "callback", "promise", "coroutine", "goroutine", "channel", "mutex", "semaphore", "deadlock", "starvation",
		"race condition", "refactoring", "debugging", "compilation", "interpretation", "virtualization", "containerization", "orchestration",
		"continuous", "integration", "delivery", "pipeline", "automation", "configuration", "provisioning", "monitoring", "logging", "tracing",
		"repository", "commit", "merge", "rebase", "branch", "checkout", "stash", "cherrypick", "conflict", "pullrequest", "review", "approval",
		"token", "session", "cookie", "storage", "indexeddb", "localstorage", "sessionstorage", "cache", "responsive", "adaptive", "mobilefirst",
		"desktopfirst", "breakpoint", "mediaquery", "flexbox", "grid", "typography", "color", "contrast", "accessibility", "internationalization",
		"localization", "serialization", "deserialization", "marshaling", "unmarshaling", "parsing", "lexing", "tokenization", "abstract syntax tree"
	]`

	defaultSynonymsDB := `{
		"fix": ["resolve", "solve", "repair", "patch", "debug", "troubleshoot", "napraw", "naprawić", "rozwiazanie", "poprawić", "rozwiazac", "workaround", "rectify", "remedy", "amend", "correct", "mend", "restore", "rebuild", "refactor", "hotfix", "tweak"],
		"error": ["bug", "issue", "glitch", "problem", "exception", "crash", "fault", "błąd", "blad", "usterka", "awaria", "wyjatek", "warning", "fail", "vulnerability", "defect", "flaw", "hiccup", "snag", "malfunction", "panic", "segfault", "timeout", "deadlock", "syntax", "runtime", "compile", "typeerror", "referenceerror"],
		"memory": ["ram", "leak", "allocation", "garbage", "pamiec", "wyciek", "heap", "stack", "cache", "buffer", "storage", "drive", "disk", "ssd", "hdd", "swap", "paging", "segmentation", "oom", "out of memory"],
		"build": ["create", "make", "develop", "write", "code", "implement", "stworz", "zbuduj", "utworz", "napisz", "tworzenie", "programowanie", "compile", "construct", "assemble", "generate", "produce", "forge", "craft", "design", "architect", "engineer"],
		"app": ["application", "software", "program", "tool", "aplikacja", "apka", "narzedzie", "system", "script", "utility", "executable", "binary", "platform", "service", "daemon", "cli", "gui", "webapp", "microservice"],
		"deploy": ["publish", "release", "host", "wdrożyć", "opublikować", "wypuścić", "production", "ci/cd", "ship", "launch", "distribute", "install", "setup", "configure", "provision", "pipeline", "rollout", "staging", "delivery", "live", "deployment"],
		"cloud": ["aws", "azure", "gcp", "docker", "kubernetes", "k8s", "serverless", "chmura", "hosting", "iaas", "paas", "saas", "compute", "instance", "vm", "container", "bucket", "cluster", "node", "pod", "lambda", "ecs", "eks"],
		"tutorial": ["guide", "how-to", "course", "lesson", "poradnik", "przewodnik", "kurs", "lekcja", "wstęp", "wprowadzenie", "walkthrough", "manual", "handbook", "primer", "bootcamp", "training", "crash course", "masterclass", "od zera", "podstawy", "step by step"],
		"doc": ["documentation", "docs", "manual", "reference", "dokumentacja", "instrukcja", "opis", "api", "spec", "specification", "readme", "wiki", "changelog", "guidelines", "swagger", "openapi", "postman", "javadoc", "godoc"],
		"learn": ["study", "understand", "master", "uczyć", "nauka", "pojac", "zrozumieć", "opanowac", "grasp", "comprehend", "acquire", "absorb", "explore", "discover", "practice", "train"],
		"example": ["sample", "demo", "przykład", "przyklad", "wzor", "snippet", "boilerplate", "template", "prototype", "mockup", "illustration", "case", "instance", "model", "repozytorium", "repository", "showcase", "usecase", "poc", "proof of concept"],
		"best": ["top", "greatest", "recommended", "optimal", "najlepszy", "najlepsze", "najlepsza", "polecane", "optymalne", "perfect", "ideal", "supreme", "ultimate", "premium", "superior", "finest", "leading", "popular", "trendy", "state of the art"],
		"fast": ["quick", "speed", "performance", "efficient", "szybki", "wydajny", "optymalizacja", "szybkość", "przyspieszyć", "lag", "latency", "swift", "rapid", "agile", "brisk", "accelerate", "boost", "throughput", "bottleneck", "benchmark", "optimized"],
		"easy": ["simple", "basic", "prosty", "latwy", "podstawowy", "beginner", "noob", "effortless", "straightforward", "uncomplicated", "clear", "accessible", "intuitive", "user-friendly", "dummy", "scratch", "trivial"],
		"lang": ["language", "język", "jezyk", "framework", "library", "biblioteka", "package", "module", "toolkit", "sdk", "api", "plugin", "extension", "addon", "dependency", "crate", "gem", "npm", "pip", "composer"],
		"db": ["database", "sql", "nosql", "baza", "danych", "storage", "query", "mysql", "postgres", "mongodb", "datastore", "repository", "warehouse", "lake", "cache", "relacyjna", "dokumentowa", "grafowa", "redis", "elastic", "sqlite", "orm"],
		"net": ["network", "web", "http", "server", "sieć", "serwer", "polaczenie", "tcp", "socket", "rest", "graphql", "internet", "intranet", "lan", "wan", "router", "switch", "firewall", "proxy", "vpn", "dns", "cdn", "load balancer", "ingress"],
		"security": ["hack", "auth", "oauth", "jwt", "token", "password", "hash", "zabezpieczenia", "hasło", "bezpieczenstwo", "encryption", "crypto", "malware", "virus", "phishing", "exploit", "patch", "xss", "csrf", "cors", "owasp", "pentest", "vulnerability", "cve", "injection"],
		"ai": ["llm", "chatgpt", "openai", "model", "neural", "prompt", "sztuczna", "inteligencja", "machine", "learning", "deep", "vision", "nlp", "generation", "inference", "training", "dataset", "gpu", "tensorflow", "pytorch", "huggingface", "transformers"],
		"test": ["testing", "unit", "e2e", "mock", "assert", "testowanie", "testy", "qa", "quality", "assurance", "validation", "verification", "coverage", "suite", "fixture", "stub", "cypress", "jest", "selenium", "integration", "playwright", "tdd", "bdd"],
		"frontend": ["ui", "ux", "client", "interfejs", "widok", "css", "html", "browser", "web", "gui", "react", "vue", "angular", "tailwind", "sass", "spa", "pwa", "dom"],
		"backend": ["server", "api", "logic", "serwer", "zaplecze", "endpoint", "microservice", "daemon", "service", "node", "django", "spring", "flask", "laravel", "express", "controller", "middleware"],
		"how": ["jak", "w jaki sposób", "sposob", "metoda", "krok po kroku", "guide", "tutorial"],
		"what": ["co", "czym", "jaki", "jaka", "ktory", "definicja", "znaczenie"],
		"why": ["dlaczego", "czemu", "powod", "przyczyna", "reason", "purpose"],
		"and": ["i", "oraz", "z", "with", "plus"],
		"mobile": ["android", "ios", "react native", "flutter", "swift", "kotlin", "xcode", "gradle", "apk", "ipa", "emulator", "simulator", "telefon", "komórka", "smartfon", "tablet"],
		"game": ["unity", "unreal", "godot", "gamedev", "gra", "silnik", "engine", "sprite", "render", "shader", "opengl", "vulkan", "directx", "physics", "collision", "fps", "multiplayer"],
		"data": ["dataset", "csv", "dataframe", "etl", "pipeline", "warehouse", "lake", "analytics", "visualization", "dashboard", "report", "chart", "metric", "kpi", "dane", "wykres", "analiza", "raport", "statystyka"],
		"devops": ["pipeline", "ci", "cd", "deploy", "infrastructure", "monitoring", "logging", "alerting", "sre", "uptime", "sla", "incident", "postmortem", "runbook", "on-call", "wdrożenie", "infrastruktura"],
		"config": ["configuration", "settings", "environment", "env", "dotenv", "yaml", "toml", "ini", "properties", "flag", "parameter", "option", "konfiguracja", "ustawienia", "parametr"],
		"perf": ["performance", "optimization", "optimize", "slow", "latency", "throughput", "bottleneck", "profiling", "benchmark", "cache", "lazy", "eager", "memoization", "wolny", "optymalizacja", "przyspieszenie"],
		"style": ["css", "styling", "layout", "responsive", "animation", "transition", "theme", "dark mode", "design system", "component library", "styl", "wygląd", "motyw", "animacja"],
		"version": ["git", "commit", "branch", "merge", "rebase", "conflict", "tag", "release", "changelog", "semver", "versioning", "wersjonowanie", "gałąź", "scalenie"]
	}`

	defaultRoutingDB := `{
		"GITHUB": ["repozytorium", "repository", "library", "biblioteka", "framework", "toolkit", "open source", "source code", "github", "projekt", "project", "example", "boilerplate", "template", "stars", "fork", "clone", "package", "module", "sdk", "plugin", "addon", "dependency", "implementation", "wrapper", "binding", "cli tool", "starter", "scaffold", "monorepo", "crate", "gem", "npm package", "pip package", "docker image", "helm chart", "github action", "extension", "demo", "sample", "awesome", "curated list", "collection", "api client", "driver", "connector", "adapter"],
		"STACK OVERFLOW": ["fix", "error", "bug", "crash", "exception", "leak", "resolve", "błąd", "naprawić", "nie działa", "fails", "null", "undefined", "syntax", "compile", "issue", "problem", "fault", "test", "testing", "auth", "how", "jak", "regex", "sql query", "join", "parsing", "format date", "convert", "cast", "type error", "syntax error", "compiler error", "runtime error", "stack trace", "debug", "troubleshoot", "workaround", "panic", "deprecat", "migration", "upgrade", "downgrade", "compatibility", "version conflict", "dependency conflict", "module not found", "import error", "cannot find", "permission denied", "access denied", "connection refused", "connection timeout", "404", "500", "cors error", "ssl error", "certificate", "encoding", "decode", "serialize", "marshal", "unmarshal", "parse json", "parse xml", "iterate", "loop through", "filter", "map reduce", "sort array", "remove duplicates", "flatten", "deep copy", "shallow copy"],
		"HACKER NEWS": ["news", "release", "update", "startup", "tech", "nowość", "aktualizacja", "wersja", "trend", "ceo", "market", "launched", "show", "ai", "security", "show hn", "ask hn", "funding", "yc", "acquisition", "merger", "opinion", "best", "vs", "alternative", "compare", "porównanie", "discus", "why", "thoughts", "experience", "future", "announcing", "ipo", "valuation", "pivot", "layoff", "hiring", "open letter", "controversy", "drama", "drama", "rant", "hot take", "unpopular opinion", "prediction", "retrospective", "postmortem", "incident", "outage", "breach", "llm", "gpt", "claude", "gemini", "copilot", "regulation", "gdpr", "eu", "antitrust", "monopoly", "disruption", "innovation"],
		"DEV.TO": ["best practices", "architecture", "tips", "tricks", "roadmap", "journey", "clean code", "solid", "design pattern", "wzorzec", "dobre praktyki", "refactoring", "tdd", "bdd", "ci/cd", "workflow", "career", "job", "interview", "resume", "portfolio", "remote", "wfh", "rekrutacja", "praca", "tutorial", "guide", "course", "poradnik", "beginner", "starter", "introduction", "getting started", "step by step", "walkthrough", "deep dive", "explained", "understand", "learn", "master", "from scratch", "zero to hero", "cheat sheet", "cheatsheet", "quick start", "handbook", "bootcamp", "basics", "fundamentals", "advanced", "intermediate", "pro tip", "productivity", "efficiency", "developer experience", "dx", "code review", "pair programming", "mob programming", "agile", "scrum", "kanban", "sprint"],
		"WIKIPEDIA": ["what is", "who is", "history", "definition", "protocol", "co to jest", "definicja", "historia", "inventor", "twórca", "algorytm", "theory", "biography", "concept", "origin", "background", "overview", "meaning", "znaczenie", "architecture", "standard", "rfc", "specification", "ieee", "iso", "w3c", "ecma", "paradigm", "taxonomy", "classification", "type system", "formal language", "computability", "complexity", "big o", "turing", "von neumann", "church", "lambda calculus", "finite automata", "context free", "halting problem", "np complete", "p vs np", "data structure", "red black tree", "b tree", "trie", "bloom filter", "skip list"]
	}`

	createIfMissing("tech_db.json", defaultTechDB)
	createIfMissing("spellcheck_db.json", defaultSpellcheckDB)
	createIfMissing("synonyms_db.json", defaultSynonymsDB)
	createIfMissing("routing_db.json", defaultRoutingDB)

	b1, _ := os.ReadFile("tech_db.json")
	var techArr []string
	_ = json.Unmarshal(b1, &techArr)
	for _, w := range techArr {
		massiveTechDB[w] = true
	}

	b2, _ := os.ReadFile("spellcheck_db.json")
	_ = json.Unmarshal(b2, &techDictionary)

	b3, _ := os.ReadFile("synonyms_db.json")
	_ = json.Unmarshal(b3, &synonymDB)

	b4, _ := os.ReadFile("routing_db.json")
	_ = json.Unmarshal(b4, &routingDB)
}

func clear() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}

func b64(s string) string {
	b, _ := base64.StdEncoding.DecodeString(s)
	return string(b)
}

func getRaw(u string) (*http.Response, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "BoltZenithApp/62.0 (Windows NT 10.0; Win64; x64) Grandmaster")
	req.Header.Set("Accept", "application/json")
	return httpClient.Do(req)
}

func levenshtein(s, t string) int {
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for j := 1; j <= len(t); j++ {
		for i := 1; i <= len(s); i++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				min := d[i-1][j]
				if d[i][j-1] < min {
					min = d[i][j-1]
				}
				if d[i-1][j-1] < min {
					min = d[i-1][j-1]
				}
				d[i][j] = min + 1
			}
		}
	}
	return d[len(s)][len(t)]
}

func autoCorrect(query string) string {
	words := strings.Fields(strings.ToLower(query))
	for i, w := range words {
		if len(w) > 3 {
			for _, dictWord := range techDictionary {
				if levenshtein(w, dictWord) == 1 {
					words[i] = dictWord
					break
				}
			}
		}
	}
	return strings.Join(words, " ")
}

func detectLanguage(query string) string {
	plIndicators := []string{"ą", "ć", "ę", "ł", "ń", "ó", "ś", "ź", "ż", "jak", "dlaczego", "co to", "jest", "nie", "się", "czy", "gdzie", "który", "dla", "przez", "albo", "więc", "tylko", "jeszcze", "teraz", "tutaj", "bardzo", "dobr", "zrób", "chcę", "potrzebuję", "pomocy", "problem", "błąd", "napraw", "szukam", "napisz", "wyjaśnij", "pokaż"}
	qLower := strings.ToLower(query)
	count := 0
	for _, ind := range plIndicators {
		if strings.Contains(qLower, ind) {
			count++
		}
	}
	if count >= 2 {
		return "pl"
	}
	for _, c := range qLower {
		if c >= 0x0400 && c <= 0x04FF {
			return "ru"
		}
		if c >= 0x4E00 && c <= 0x9FFF {
			return "zh"
		}
		if c >= 0x3040 && c <= 0x309F {
			return "ja"
		}
		if c >= 0xAC00 && c <= 0xD7AF {
			return "ko"
		}
		if c >= 0x00C0 && c <= 0x00FF {
			deIndicators := []string{"ü", "ö", "ä", "ß", "wie", "warum", "ist", "nicht", "haben", "werden"}
			for _, di := range deIndicators {
				if strings.Contains(qLower, di) {
					return "de"
				}
			}
			esIndicators := []string{"ñ", "cómo", "por qué", "qué", "para", "hacer", "desde", "también", "puede"}
			for _, ei := range esIndicators {
				if strings.Contains(qLower, ei) {
					return "es"
				}
			}
			frIndicators := []string{"è", "ê", "ç", "comment", "pourquoi", "qu'est", "faire", "avoir", "être"}
			for _, fi := range frIndicators {
				if strings.Contains(qLower, fi) {
					return "fr"
				}
			}
		}
	}
	return "en"
}

func translateToEnglish(query string) string {
	lang := detectLanguage(query)
	if lang == "en" {
		return query
	}

	libreURL := "https://libretranslate.de/translate"
	payload := fmt.Sprintf(`{"q":%q,"source":"%s","target":"en","format":"text"}`, query, lang)
	req, err := http.NewRequest("POST", libreURL, strings.NewReader(payload))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "BoltZenithApp/62.0")
		resp, err := httpClient.Do(req)
		if err == nil && resp.StatusCode == 200 {
			var libreRes struct {
				TranslatedText string `json:"translatedText"`
			}
			if json.NewDecoder(resp.Body).Decode(&libreRes) == nil && libreRes.TranslatedText != "" {
				resp.Body.Close()
				return strings.ToLower(libreRes.TranslatedText)
			}
			resp.Body.Close()
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	u := "https://api.mymemory.translated.net/get?q=" + url.QueryEscape(query) + "&langpair=" + lang + "|en"
	resp, err := getRaw(u)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return query
	}
	defer resp.Body.Close()

	var res struct {
		ResponseData struct {
			TranslatedText string `json:"translatedText"`
		} `json:"responseData"`
	}
	if json.NewDecoder(resp.Body).Decode(&res) == nil && res.ResponseData.TranslatedText != "" {
		translated := strings.ToLower(res.ResponseData.TranslatedText)
		if translated != strings.ToLower(query) {
			return translated
		}
	}

	aiPrompt := `Translate the following text to English for a search engine. Output ONLY the translation. If it's already English, return it as is.
Text: ` + query
	aiTranslated := fetchSimpleAI(aiPrompt)
	if aiTranslated != "" && len(aiTranslated) < len(query)*5 {
		return strings.ToLower(aiTranslated)
	}

	return query
}

func getVQD() string {
	req, _ := http.NewRequest("GET", "https://duckduckgo.com/duckchat/v1/status", nil)
	req.Header.Set("x-vqd-accept", "1")
	req.Header.Set("User-Agent", "BoltZenithApp/62.0")
	resp, err := httpClient.Do(req)
	if err == nil && resp.StatusCode == 200 {
		vqd := resp.Header.Get("x-vqd-4")
		resp.Body.Close()
		return vqd
	}
	if resp != nil {
		resp.Body.Close()
	}
	return ""
}

func fetchSimpleAI(prompt string) string {
	// 1. Pollinations AI (POST for reliability)
	plPayload := fmt.Sprintf(`{"messages":[{"role":"user","content":%q}],"model":"openai","json":false}`, prompt)
	plReq, _ := http.NewRequest("POST", "https://text.pollinations.ai/", strings.NewReader(plPayload))
	plReq.Header.Set("Content-Type", "application/json")
	plResp, err := httpClient.Do(plReq)
	if err == nil && plResp.StatusCode == 200 {
		body, readErr := io.ReadAll(plResp.Body)
		plResp.Body.Close()
		if readErr == nil && len(body) > 0 {
			return cleanHTMLAndMarkdown(string(body), false)
		}
	}
	if plResp != nil {
		plResp.Body.Close()
	}

	// 2. DuckDuckGo AI (Token-based)
	vqd := getVQD()
	if vqd != "" {
		ddgPayload := fmt.Sprintf(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":%q}]}`, prompt)
		ddgReq, _ := http.NewRequest("POST", "https://duckduckgo.com/duckchat/v1/chat", strings.NewReader(ddgPayload))
		ddgReq.Header.Set("Content-Type", "application/json")
		ddgReq.Header.Set("x-vqd-4", vqd)
		ddgReq.Header.Set("User-Agent", "BoltZenithApp/62.0")
		ddgResp, err := httpClient.Do(ddgReq)
		if err == nil && ddgResp.StatusCode == 200 {
			scanner := bufio.NewScanner(ddgResp.Body)
			var sb strings.Builder
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					if data == "[DONE]" {
						break
					}
					var chunk struct {
						Message string `json:"message"`
					}
					json.Unmarshal([]byte(data), &chunk)
					sb.WriteString(chunk.Message)
				}
			}
			ddgResp.Body.Close()
			if sb.Len() > 0 {
				return cleanHTMLAndMarkdown(sb.String(), false)
			}
		}
		if ddgResp != nil {
			ddgResp.Body.Close()
		}
	}

	return ""
}

func classifyComplexity(q string) string {
	q = strings.ToLower(q)
	words := strings.Fields(q)

	if len(words) <= 3 {
		isTech := false
		for _, w := range words {
			if massiveTechDB[w] {
				isTech = true
				break
			}
		}
		if !isTech {
			return "EASY"
		}
	}

	if len(words) > 10 || strings.Contains(q, "ponieważ") || strings.Contains(q, "dlatego") || strings.Contains(q, "problem") || strings.Contains(q, "error") {
		return "HARD"
	}

	return "COMPLEX"
}

func manualTranslateMenu(reader *bufio.Reader, query string) (string, string) {
	fmt.Print("\033[35m❓ Session Language (Press Enter for No-Translate or Target Lang Code: en, pl, de, fr, es): \033[0m")
	ans, _ := reader.ReadString('\n')
	ans = strings.ToLower(strings.TrimSpace(ans))

	if ans == "" {
		return query, ""
	}

	targetLang := ""
	switch ans {
	case "1", "en", "english":
		targetLang = "English"
	case "2", "pl", "polish", "polski":
		targetLang = "Polish"
	case "3", "de", "german":
		targetLang = "German"
	case "4", "fr", "french":
		targetLang = "French"
	case "5", "es", "spanish":
		targetLang = "Spanish"
	default:
		return query, ""
	}

	translated := query
	if targetLang != "English" {
		fmt.Printf("\033[35m🧠 Optimizing query for %s results...\033[0m\n", targetLang)
		prompt := fmt.Sprintf("Translate for search query to %s. Output ONLY translated keywords. Query: %s", targetLang, query)
		translated = fetchSimpleAI(prompt)
		if translated == "" {
			translated = query
		}
	} else {
		fmt.Println("\033[35m🧠 Using English session mode.\033[0m")
	}

	return translated, targetLang
}

func fastTranslate(q string) string {
	low := strings.ToLower(q)
	dict := map[string]string{
		"siema":         "hello",
		"jaki":          "what",
		"kto":           "who",
		"popularny":     "popular",
		"hacker":        "hacker",
		"jest":          "is",
		"stolica":       "capital",
		"polski":        "poland",
		"błąd":          "error",
		"napraw":        "fix",
		"jak":           "how",
		"dlaczego":      "why",
		"kiedy":         "when",
		"gdzie":         "where",
		"pomoc":         "help",
		"stworzyc":      "create",
		"stworzyć":      "create",
		"pobierz":       "download",
		"najlepszy":     "best",
		"porownaj":      "compare",
		"historia":      "history",
		"polska":        "poland",
		"angielski":     "english",
		"niemiecki":     "german",
		"niemcy":        "germany",
		"francja":       "france",
		"hiszpania":     "spain",
		"wlochy":        "italy",
		"język":         "language",
		"programowanie": "programming",
		"kod":           "code",
		"strona":        "website",
		"aplikacja":     "app",
		"szukaj":        "search",
		"pieniadze":     "money",
		"pogoda":        "weather",
		"wiadomosci":    "news",
		"sport":         "sport",
	}

	words := strings.Fields(rePunct.ReplaceAllString(low, " "))
	var res []string
	for _, w := range words {
		if tr, ok := dict[w]; ok {
			res = append(res, tr)
		} else {
			res = append(res, w)
		}
	}
	return strings.Join(res, " ")
}

func classifyCategory(query string) string {
	q := strings.ToLower(query)

	countries := []string{"poland", "polska", "germany", "france", "usa", "ukraine", "italy", "spain", "stolica", "capital", "kraj", "country", "warszawa", "berlin", "londyn", "london"}
	for _, c := range countries {
		if strings.Contains(q, c) {
			return "GEOPOLITICAL"
		}
	}

	techKeywords := []string{"code", "program", "api", "error", "bug", "fix", "develop", "software", "hardware", "cpu", "server", "linux", "database"}
	for _, tk := range techKeywords {
		if strings.Contains(q, tk) {
			return "TECHNICAL"
		}
	}
	for w := range massiveTechDB {
		if strings.Contains(q, w) {
			return "TECHNICAL"
		}
	}

	bioKeywords := []string{"who is", "kto to", "kim był", "biografia", "biography", "born", "urodzony", "życiorys"}
	for _, k := range bioKeywords {
		if strings.Contains(q, k) {
			return "BIOGRAPHY"
		}
	}

	scienceKeywords := []string{"nauka", "science", "nature", "space", "planet", "physics", "biology", "chemistry", "research", "badania"}
	for _, sk := range scienceKeywords {
		if strings.Contains(q, sk) {
			return "SCIENCE"
		}
	}

	gamingKeywords := []string{"game", "gra", "multiplayer", "ps5", "xbox", "nintendo", "steam", "fps", "rpg"}
	for _, gk := range gamingKeywords {
		if strings.Contains(q, gk) {
			return "GAMING"
		}
	}

	return "GENERAL"
}

func classifyQueryIntent(query string) map[string]int {
	qLower := strings.ToLower(query)
	boost := map[string]int{}

	errorPatterns := []string{"error", "bug", "fix", "crash", "fail", "exception", "not working", "broken", "issue", "problem", "cannot", "unable", "doesn't work", "won't", "throws", "panic", "segfault", "null", "undefined", "nan", "timeout", "rejected", "denied", "refused", "błąd", "nie działa", "napraw", "awaria", "wyjatek"}
	howtoPatterns := []string{"how to", "how do", "how can", "jak", "tutorial", "guide", "step by step", "example", "implement", "create", "build", "setup", "install", "configure", "connect", "integrate", "use", "write", "make", "set up"}
	comparePatterns := []string{"vs", "versus", "compare", "comparison", "difference", "better", "best", "alternative", "pros cons", "benchmark", "which one", "porównanie", "porównaj", "najlepszy", "lepszy"}
	learnPatterns := []string{"what is", "what are", "explain", "definition", "meaning", "concept", "theory", "history", "overview", "introduction", "co to", "czym jest", "definicja", "wyjaśnij", "znaczenie"}
	findPatterns := []string{"library", "framework", "tool", "package", "plugin", "module", "sdk", "repo", "repository", "open source", "find", "looking for", "recommend", "suggestion", "szukam", "polecam", "biblioteka", "narzędzie"}
	newsPatterns := []string{"news", "release", "update", "latest", "new version", "announcement", "launched", "trending", "opinion", "thoughts", "future", "startup", "funding", "nowość", "aktualizacja", "premiera"}
	careerPatterns := []string{"job", "career", "interview", "resume", "salary", "hire", "remote", "junior", "senior", "portfolio", "praca", "rekrutacja", "rozmowa", "kariera", "wynagrodzenie"}

	for _, p := range errorPatterns {
		if strings.Contains(qLower, p) {
			boost["STACK OVERFLOW"] += 20
			boost["GITHUB"] += 8
		}
	}
	for _, p := range howtoPatterns {
		if strings.Contains(qLower, p) {
			boost["STACK OVERFLOW"] += 15
			boost["DEV.TO"] += 12
			boost["GITHUB"] += 5
		}
	}
	for _, p := range comparePatterns {
		if strings.Contains(qLower, p) {
			boost["HACKER NEWS"] += 18
			boost["DEV.TO"] += 15
		}
	}
	for _, p := range learnPatterns {
		if strings.Contains(qLower, p) {
			boost["WIKIPEDIA"] += 25
			boost["DEV.TO"] += 8
		}
	}
	for _, p := range findPatterns {
		if strings.Contains(qLower, p) {
			boost["GITHUB"] += 25
			boost["HACKER NEWS"] += 5
		}
	}
	for _, p := range newsPatterns {
		if strings.Contains(qLower, p) {
			boost["HACKER NEWS"] += 22
			boost["DEV.TO"] += 10
		}
	}
	for _, p := range careerPatterns {
		if strings.Contains(qLower, p) {
			boost["DEV.TO"] += 20
			boost["HACKER NEWS"] += 10
		}
	}

	return boost
}

func generateMultiQueries(brain BrainResult) []string {
	baseQuery := brain.OptimizedQ
	queries := []string{baseQuery}
	words := strings.Fields(baseQuery)

	if brain.Primary != "" {
		// Strict primary search
		sq := "\"" + brain.Primary + "\" " + brain.Action + " " + brain.Context
		if brain.Exclude != "" {
			sq += " -" + brain.Exclude
		}
		queries = append(queries, sq)
		if brain.Action != "" {
			// Extreme precision: Primary + Action in quotes
			aq := "\"" + brain.Primary + "\" \"" + brain.Action + "\""
			if brain.Exclude != "" {
				aq += " -" + brain.Exclude
			}
			queries = append(queries, aq)
		}
	} else if len(words) > 1 {
		queries = append(queries, "\""+baseQuery+"\"")
	}

	lowQ := strings.ToLower(baseQuery)
	if strings.Contains(lowQ, "how to") || strings.Contains(lowQ, "fix") || strings.Contains(lowQ, "error") {
		queries = append(queries, baseQuery+" solved")
		queries = append(queries, baseQuery+" best practices")
	} else if strings.Contains(lowQ, "vs") || strings.Contains(lowQ, "best") || strings.Contains(lowQ, "compare") {
		queries = append(queries, baseQuery+" architecture comparison")
		queries = append(queries, baseQuery+" performance benchmark")
	}

	return queries
}

func generateGitHubQueries(baseQuery string) []string {
	queries := []string{baseQuery}
	words := strings.Fields(strings.ToLower(baseQuery))
	var techWords []string
	var actionWords []string
	for _, w := range words {
		if massiveTechDB[w] {
			techWords = append(techWords, w)
		} else if len(w) > 3 {
			actionWords = append(actionWords, w)
		}
	}
	if len(techWords) > 0 {
		techQ := strings.Join(techWords, " ")
		if techQ != baseQuery {
			queries = append(queries, techQ)
		}
		if len(actionWords) > 0 {
			combined := strings.Join(techWords, " ") + " " + strings.Join(actionWords, " ")
			if combined != baseQuery {
				queries = append(queries, combined)
			}
		}
	}
	return queries
}

func buildGitHubQuery(raw string) string {
	q := strings.ToLower(raw)
	q = rePunct.ReplaceAllString(q, " ")
	words := strings.Fields(q)
	ghStopwords := map[string]bool{
		"how": true, "to": true, "what": true, "is": true, "the": true, "a": true, "an": true,
		"of": true, "jak": true, "co": true, "dlaczego": true, "w": true, "z": true,
		"i": true, "oraz": true, "do": true, "na": true, "for": true, "and": true,
		"this": true, "that": true, "it": true, "or": true, "be": true, "are": true,
		"by": true, "from": true, "can": true, "which": true, "their": true,
		"my": true, "your": true, "me": true, "we": true, "they": true,
		"has": true, "have": true, "had": true, "was": true, "were": true,
		"been": true, "being": true, "will": true, "would": true, "should": true,
		"could": true, "may": true, "might": true, "shall": true, "must": true,
		"need": true, "want": true, "like": true, "just": true, "about": true,
		"jaki": true, "jaka": true, "jakie": true, "który": true, "ktora": true,
		"o": true, "przy": true, "dla": true, "taki": true, "take": true,
		"pod": true, "nad": true, "za": true, "po": true, "przez": true,
	}
	var techWords []string
	var otherWords []string
	for _, w := range words {
		if ghStopwords[w] {
			continue
		}
		if massiveTechDB[w] {
			techWords = append(techWords, w)
		} else {
			otherWords = append(otherWords, w)
		}
	}
	var apiWords []string
	apiWords = append(apiWords, techWords...)
	apiWords = append(apiWords, otherWords...)
	if len(apiWords) == 0 {
		return raw
	}
	if len(apiWords) > 6 {
		apiWords = apiWords[:6]
	}
	return strings.Join(apiWords, " ")
}

func extractCode(text string) (string, string) {
	codeSnippet := ""
	matches := reCodeBlock.FindStringSubmatch(text)
	if len(matches) > 1 {
		codeSnippet = strings.TrimSpace(matches[1])
	}
	cleanText := reCodeBlock.ReplaceAllString(text, "")
	return codeSnippet, cleanText
}

func formatForTerminal(s string) string {
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&amp;", "&")

	s = reMDBold.ReplaceAllString(s, ansiBold)

	s = reMDHeaders.ReplaceAllString(s, ansiBold+ansiCyan)

	lines := strings.Split(s, "\n")
	inCode := false
	var final []string
	for _, l := range lines {
		if strings.HasPrefix(l, "```") {
			inCode = !inCode
			if inCode {
				final = append(final, ansiGreen+"[CODE BLOCK START]")
			} else {
				final = append(final, "[CODE BLOCK END]"+ansiReset)
			}
			continue
		}
		if inCode {
			final = append(final, "  "+ansiGreen+l+ansiReset)
		} else {
			final = append(final, l)
		}
	}
	s = strings.Join(final, "\n")

	return s + ansiReset
}

func cleanHTMLAndMarkdown(s string, keepNewlines bool) string {
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#x27;", "'")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = reHTML.ReplaceAllString(s, "")
	s = reMDLinks.ReplaceAllString(s, "$1")
	s = reMDHeaders.ReplaceAllString(s, "")
	s = reMDBold.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "```", "")
	if !keepNewlines {
		s = strings.Join(strings.Fields(s), " ")
	} else {
		s = reNL.ReplaceAllString(s, "\n\n")
	}
	return strings.TrimSpace(s)
}

func buildAPIQuery(raw string) string {
	q := strings.ToLower(raw)
	q = rePunct.ReplaceAllString(q, " ")
	words := strings.Fields(q)

	stopwords := map[string]bool{
		"how": true, "to": true, "what": true, "is": true, "the": true, "a": true, "an": true,
		"of": true, "in": true, "on": true, "with": true, "jak": true, "co": true, "dlaczego": true,
		"w": true, "z": true, "i": true, "oraz": true, "do": true, "na": true, "my": true,
		"your": true, "me": true, "we": true, "they": true, "it": true, "this": true,
		"that": true, "for": true, "and": true, "or": true, "but": true, "be": true,
		"are": true, "can": true, "will": true, "would": true, "should": true, "about": true,
		"just": true, "like": true, "some": true, "very": true, "there": true, "here": true,
		"when": true, "where": true, "who": true, "which": true, "been": true, "any": true,
		"ma": true, "się": true, "za": true, "po": true, "przez": true, "pod": true, "nad": true,
		"jaki": true, "jaka": true, "jakie": true, "który": true, "która": true, "o": true,
		"przy": true, "dla": true, "taki": true, "take": true, "ten": true, "ta": true,
		"by": true, "from": true, "into": true, "onto": true, "towards": true, "away": true,
		"up": true, "down": true, "left": true, "right": true, "top": true, "bottom": true,
		"middle": true, "side": true, "under": true, "over": true, "between": true, "among": true,
		"through": true, "against": true, "during": true, "before": true, "after": true, "since": true,
		"until": true, "because": true, "as": true, "so": true,
		"unless": true, "though": true, "although": true, "even": true, "if": true, "else": true,
		"than": true, "more": true, "less": true, "most": true, "least": true, "better": true,
		"worse": true, "well": true, "badly": true, "hard": true, "easy": true, "simply": true,
		"clearly": true, "usually": true, "often": true, "sometimes": true, "never": true, "always": true,
		"ever": true, "only": true, "almost": true, "barely": true,
		"nearly": true, "hardly": true, "scarcely": true, "quite": true, "rather": true, "pretty": true,
		"fairly": true, "somewhat": true, "total": true, "completely": true, "totally": true, "utterly": true,
		"entirely": true, "absolutely": true, "fully": true, "partially": true, "partly": true, "mostly": true,
		"mainly": true, "chiefly": true, "generally": true, "roughly": true, "approximately": true,
	}

	var techWords []string
	var conceptWords []string
	var normalWords []string

	for _, w := range words {
		if stopwords[w] {
			continue
		}

		isTech := massiveTechDB[w]
		normalized := w
		for key, syns := range synonymDB {
			if w == key {
				normalized = key
				break
			}
			found := false
			for _, s := range syns {
				if w == s {
					normalized = key
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if isTech {
			techWords = append(techWords, normalized)
		} else if normalized != w {
			conceptWords = append(conceptWords, normalized)
		} else if len(w) > 2 {
			normalWords = append(normalWords, w)
		}
	}

	var apiWords []string
	apiWords = append(apiWords, techWords...)
	apiWords = append(apiWords, conceptWords...)
	apiWords = append(apiWords, normalWords...)

	if len(apiWords) == 0 {
		return raw
	}

	seen := make(map[string]bool)
	var deduped []string
	for _, w := range apiWords {
		if !seen[w] {
			seen[w] = true
			deduped = append(deduped, w)
		}
	}

	if len(deduped) > 8 {
		deduped = deduped[:8]
	}

	return strings.Join(deduped, " ")
}

func internalDetermineTarget(query string, useAI bool) []string {
	qLower := strings.ToLower(query)
	qLowerClean := rePunct.ReplaceAllString(qLower, " ")
	words := strings.Fields(qLowerClean)

	techWordCount := 0
	for _, w := range words {
		if massiveTechDB[w] {
			techWordCount++
			continue
		}
		for key, syns := range synonymDB {
			if w == key {
				techWordCount++
				break
			}
			for _, syn := range syns {
				if w == syn {
					techWordCount++
					break
				}
			}
		}
	}

	scores := map[string]int{
		"GITHUB":         0,
		"STACK OVERFLOW": 0,
		"HACKER NEWS":    0,
		"DEV.TO":         0,
		"WIKIPEDIA":      0,
	}

	if useAI {
		if techWordCount == 0 {
			fmt.Println("\033[35mBrak słów kluczowych IT. Analizowanie przez AI...\033[0m")
			aiScores := getAITargetScores(query)
			for k, v := range aiScores {
				scores[k] += v
			}
		} else {
			aiScores := getAITargetScores(query)
			for k, v := range aiScores {
				scores[k] += v / 2
			}
		}
	}

	if techWordCount > 0 {
		for target, keywords := range routingDB {
			for _, kw := range keywords {
				if kw == "" {
					continue
				}
				if strings.Contains(" "+qLowerClean+" ", " "+kw+" ") {
					scores[target] += 15
				} else if strings.Contains(qLower, kw) {
					scores[target] += 8
				}
			}
		}
	}

	intentBoost := classifyQueryIntent(query)
	for target, boost := range intentBoost {
		scores[target] += boost
	}

	type kv struct {
		Key   string
		Value int
	}
	var sortedScores []kv
	for k, v := range scores {
		sortedScores = append(sortedScores, kv{k, v})
	}
	sort.Slice(sortedScores, func(i, j int) bool {
		if sortedScores[i].Value == sortedScores[j].Value {
			return sortedScores[i].Key < sortedScores[j].Key
		}
		return sortedScores[i].Value > sortedScores[j].Value
	})

	var bestTargets []string
	if sortedScores[0].Value == 0 {
		return []string{"STACK OVERFLOW", "GITHUB"}
	}

	bestTargets = append(bestTargets, sortedScores[0].Key)

	if sortedScores[1].Value > 0 && sortedScores[1].Value >= sortedScores[0].Value*4/10 {
		bestTargets = append(bestTargets, sortedScores[1].Key)
	}

	return bestTargets
}

func determineTarget(query string) []string {
	return internalDetermineTarget(query, true)
}

func determineTargetLocal(query string) []string {
	return internalDetermineTarget(query, false)
}

func calculateScore(rawQuery string, title string, body string, source string, priority int) int {
	score := 50 + (priority * 3)
	qLower := strings.ToLower(rawQuery)
	tLower := strings.ToLower(title)
	bLower := strings.ToLower(body)
	wordsRaw := strings.Fields(qLower)
	var negativeWords []string
	var positiveWords []string
	for _, w := range wordsRaw {
		if strings.HasPrefix(w, "-") && len(w) > 1 {
			negativeWords = append(negativeWords, w[1:])
		} else {
			positiveWords = append(positiveWords, w)
		}
	}

	for _, neg := range negativeWords {
		if strings.Contains(tLower, neg) || strings.Contains(bLower, neg) {
			return -10000
		}
	}

	qClean := rePunct.ReplaceAllString(strings.Join(positiveWords, " "), " ")
	tClean := rePunct.ReplaceAllString(tLower, " ")

	optWords := strings.Fields(qClean)
	exactMatch := true
	for _, ow := range optWords {
		if len(ow) < 3 {
			continue
		}

		paddedT := " " + tClean + " "
		wordMatch := strings.Contains(paddedT, " "+ow+" ")

		if wordMatch {
			score += 6500 + (priority * 20)
			if massiveTechDB[ow] {
				score += 3000
			}
		} else if strings.Contains(tClean, ow) {
			score += 1500
		} else {
			exactMatch = false
		}

		if strings.Contains(bLower, ow) {
			score += 800 + (priority * 5)
		}
	}

	titleWords := strings.Fields(tClean)
	if len(titleWords) > len(optWords) && len(optWords) > 0 {
		extraWords := len(titleWords) - len(optWords)
		score -= extraWords * 1200
	}

	if exactMatch && len(optWords) > 1 {
		score += 18000
	}

	if len(optWords) >= 2 {
		for i := 0; i < len(optWords)-1; i++ {
			bigram := optWords[i] + " " + optWords[i+1]
			if strings.Contains(tClean, bigram) {
				score += 8000
			} else if strings.Contains(bLower, bigram) {
				score += 3500
			}
		}
	}

	if strings.Contains(bLower, "http") {
		for _, w := range optWords {
			if len(w) > 3 && strings.Contains(bLower, "/"+w) {
				score += 1500
			}
		}
	}

	isErrorQuery := strings.Contains(qLower, "error") || strings.Contains(qLower, "bug") || strings.Contains(qLower, "błąd") || strings.Contains(qLower, "fix") || strings.Contains(qLower, "crash") || strings.Contains(qLower, "failed")
	isHowToQuery := strings.Contains(qLower, "how to") || strings.Contains(qLower, "jak") || strings.Contains(qLower, "tutorial") || strings.Contains(qLower, "guide")
	isBestQuery := strings.Contains(qLower, "best") || strings.Contains(qLower, "vs") || strings.Contains(qLower, "compare") || strings.Contains(qLower, "alternative")
	isWhatQuery := strings.Contains(qLower, "what is") || strings.Contains(qLower, "co to") || strings.Contains(qLower, "definition") || strings.Contains(qLower, "meaning")

	if isErrorQuery {
		if source == "STACK OVERFLOW" {
			score += 600
		} else if source == "GITHUB" {
			score += 400
		}
	}
	if isHowToQuery {
		if source == "STACK OVERFLOW" || source == "DEV.TO" {
			score += 450
		}
		if strings.Contains(tLower, "how to") || strings.Contains(tLower, "guide") {
			score += 1200
		}
	}
	if isBestQuery {
		if source == "HACKER NEWS" || source == "DEV.TO" {
			score += 500
		}
	}
	if isWhatQuery {
		if source == "WIKIPEDIA" {
			score += 800
		}
	}

	if tClean == qClean {
		score += 5000
	} else if strings.Contains(tLower, qLower) {
		score += 1800
	}

	words := optWords
	validWords := 0
	titleMatches := 0
	bodyMatches := 0

	for _, w := range words {
		if len(w) < 3 {
			continue
		}

		matchedInTitle := strings.Contains(tClean, w)
		matchedInBody := strings.Contains(bLower, w)

		isTech := massiveTechDB[w]
		isGeneric := map[string]bool{"error": true, "bug": true, "fix": true, "api": true, "code": true, "tool": true, "app": true, "jak": true, "how": true}

		wordWeight := 150
		bodyWeight := 45

		if isTech {
			if isGeneric[w] {
				wordWeight = 300
			} else {
				wordWeight = 850
			}
			bodyWeight = 220
		}

		if !matchedInTitle || !matchedInBody {
			if syns, ok := synonymDB[w]; ok {
				for _, syn := range syns {
					if !matchedInTitle && strings.Contains(tClean, syn) {
						matchedInTitle = true
						score += wordWeight / 2
					}
					if !matchedInBody && strings.Contains(bLower, syn) {
						matchedInBody = true
						score += bodyWeight / 2
					}
				}
			}
		}

		if matchedInTitle {
			score += wordWeight
			titleMatches++
		}
		if matchedInBody {
			score += bodyWeight
			bodyMatches++
		}

		acronyms := map[string]string{
			"js":   "javascript",
			"ts":   "typescript",
			"py":   "python",
			"rb":   "ruby",
			"rs":   "rust",
			"go":   "golang",
			"k8s":  "kubernetes",
			"ai":   "artificial intelligence",
			"ml":   "machine learning",
			"db":   "database",
			"api":  "application programming interface",
			"os":   "operating system",
			"cli":  "command line interface",
			"gui":  "graphical user interface",
			"iot":  "internet of things",
			"ci":   "continuous integration",
			"cd":   "continuous deployment",
			"rest": "representational state transfer",
			"crud": "create read update delete",
			"jvm":  "java virtual machine",
			"dom":  "document object model",
			"html": "hypertext markup language",
			"css":  "cascading style sheets",
			"sql":  "structured query language",
		}
		if full, ok := acronyms[w]; ok {
			if strings.Contains(tLower, full) || strings.Contains(bLower, full) {
				score += 3000
			}
		}

		validWords++
	}

	currentYear := time.Now().Year()
	for y := currentYear; y >= currentYear-2; y-- {
		yearStr := fmt.Sprintf("%d", y)
		if strings.Contains(tLower, yearStr) || strings.Contains(bLower, yearStr) {
			score += 500
			break
		}
	}

	trustedDomains := []string{"docs.", "developer.", "blog.golang", "react.dev", "github.com", "stackoverflow.com", "mozilla.org", "medium.com/engineering", "dev.to"}
	for _, domain := range trustedDomains {
		if strings.Contains(bLower, domain) {
			score += 350
		}
	}

	if validWords > 0 {
		if titleMatches == validWords {
			score += 1500
		} else if float64(titleMatches)/float64(validWords) > 0.7 {
			score += 600
		}
		if titleMatches == 0 && bodyMatches == 0 {
			score -= 3000
		} else if titleMatches == 0 {
			score -= 500
		}
	}

	category := classifyCategory(rawQuery)
	translatedQ := strings.ToLower(fastTranslate(rawQuery))

	if category == "GEOPOLITICAL" {

		isExactCountry := (tLower == qLower || tLower == translatedQ || tLower == "poland")
		if isExactCountry {
			score += 40000
		}

		if source == "WIKIPEDIA" {
			if !isExactCountry && strings.Contains(tLower, " ") {
				score -= 8000
			}
		}
	}

	if category == "TECHNICAL" {
		if source == "STACK OVERFLOW" || source == "GITHUB" {
			score += 2000
		}
	}

	if category == "BIOGRAPHY" {
		if source == "WIKIPEDIA" {
			score += 3000
		}
	}

	if strings.Contains(tLower, "disambiguation") || strings.Contains(tLower, "ujednoznacznienie") || strings.Contains(tLower, "list of") {
		score -= 10000
	}

	return score
}

type gitHubItem struct {
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	HtmlUrl     string   `json:"html_url"`
	Language    string   `json:"language"`
	Stars       int      `json:"stargazers_count"`
	Forks       int      `json:"forks_count"`
	Topics      []string `json:"topics"`
	License     *struct {
		Name string `json:"name"`
	} `json:"license"`
	UpdatedAt  string `json:"updated_at"`
	Archived   bool   `json:"archived"`
	OpenIssues int    `json:"open_issues_count"`
}

func fetchGitHubSearch(queryStr string, sortBy string) []gitHubItem {
	baseURL := b64("aHR0cHM6Ly9hcGkuZ2l0aHViLmNvbS9zZWFyY2gvcmVwb3NpdG9yaWVzP3E9")
	u := baseURL + url.QueryEscape(queryStr) + "&per_page=10"
	if sortBy != "" {
		u += "&sort=" + sortBy + "&order=desc"
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "BoltZenithApp/62.0")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()

	var res struct {
		Items []gitHubItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil
	}
	return res.Items
}

func scoreGitHubItem(rawQ string, item gitHubItem) int {
	title := strings.ToLower(item.FullName)
	desc := strings.ToLower(item.Description)
	qLower := strings.ToLower(rawQ)
	qClean := rePunct.ReplaceAllString(qLower, " ")
	words := strings.Fields(qClean)

	score := 50

	repoName := title
	if idx := strings.Index(title, "/"); idx >= 0 {
		repoName = title[idx+1:]
	}

	nameClean := rePunct.ReplaceAllString(repoName, " ")
	nameParts := strings.Fields(nameClean)

	joinedWords := strings.ReplaceAll(strings.Join(words, ""), " ", "")
	if strings.ReplaceAll(repoName, "-", "") == joinedWords {
		score += 6500
	} else if repoName == strings.ReplaceAll(qClean, " ", "-") {
		score += 5500
	} else if strings.Contains(repoName, strings.ReplaceAll(qClean, " ", "-")) {
		score += 4000
	} else if strings.Contains(repoName, strings.ReplaceAll(qClean, " ", "")) {
		score += 3000
	}

	nameMatches := 0
	for _, w := range words {
		if len(w) < 2 {
			continue
		}
		for _, np := range nameParts {
			if w == np {
				nameMatches++
				score += 400
				break
			} else if strings.Contains(np, w) || strings.Contains(w, np) {
				nameMatches++
				score += 200
				break
			}
		}
	}
	if len(words) > 0 && nameMatches == len(words) {
		score += 2500
	} else if len(words) > 1 && float64(nameMatches)/float64(len(words)) > 0.7 {
		score += 1200
	}

	descMatches := 0
	for _, w := range words {
		if len(w) < 3 {
			continue
		}
		if strings.Contains(desc, w) {
			descMatches++
			score += 150
		}
	}
	if len(words) > 0 && descMatches == len(words) {
		score += 800
	}

	topicMatches := 0
	for _, w := range words {
		if len(w) < 2 {
			continue
		}
		for _, topic := range item.Topics {
			tl := strings.ToLower(topic)
			if tl == w {
				topicMatches++
				score += 500
				break
			} else if strings.Contains(tl, w) {
				topicMatches++
				score += 200
				break
			}
		}
	}

	for _, w := range words {
		if len(w) < 2 {
			continue
		}
		langLower := strings.ToLower(item.Language)
		if langLower == w || strings.Contains(langLower, w) {
			score += 800
		}
	}

	if item.License != nil && item.License.Name != "" {
		score += 450
	}
	if strings.Contains(title, "awesome") || strings.Contains(desc, "awesome") {
		score += 2000
	}
	if strings.Contains(desc, "official") || strings.Contains(desc, "maintained") || strings.Contains(desc, "production") || strings.Contains(desc, "verified") {
		score += 1000
	}
	if strings.HasPrefix(title, "google/") || strings.HasPrefix(title, "microsoft/") || strings.HasPrefix(title, "facebook/") || strings.HasPrefix(title, "golang/") {
		score += 1500
	}

	if item.Stars > 100000 {
		score += 1200
	} else if item.Stars > 50000 {
		score += 1000
	} else if item.Stars > 10000 {
		score += 800
	} else if item.Stars > 5000 {
		score += 600
	} else if item.Stars > 1000 {
		score += 400
	} else if item.Stars > 500 {
		score += 250
	} else if item.Stars > 100 {
		score += 150
	}

	if item.Stars > 0 && item.Forks > 0 {
		ratio := float64(item.Forks) / float64(item.Stars)
		if ratio > 0.15 && ratio < 0.6 {
			score += 200
		}
	}

	if item.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, item.UpdatedAt); err == nil {
			days := int(time.Since(t).Hours() / 24)
			if days < 7 {
				score += 500
			} else if days < 30 {
				score += 350
			} else if days < 90 {
				score += 200
			} else if days < 365 {
				score += 100
			} else if days > 730 {
				score -= 500
			} else if days > 1825 {
				score -= 1500
			}
		}
	}

	if item.Archived {
		score -= 2000
	}

	if item.OpenIssues > 1000 {
		score -= 100
	}

	if nameMatches == 0 && descMatches == 0 && topicMatches == 0 {
		score -= 2500
	} else if nameMatches == 0 && descMatches == 0 {
		score -= 800
	}

	return score
}

func fetchGitHub(rawQ string, ch chan<- BoltResult, wg *sync.WaitGroup, priority int) {
	defer wg.Done()
	ghQ := buildGitHubQuery(rawQ)

	type searchJob struct {
		query  string
		sortBy string
	}

	jobs := []searchJob{
		{ghQ + " in:name", "stars"},
		{ghQ + " in:name,description", "stars"},
		{ghQ, ""},
	}

	var subWg sync.WaitGroup
	var mu sync.Mutex
	seenRepos := make(map[string]bool)
	var allResults []BoltResult

	for _, job := range jobs {
		subWg.Add(1)
		go func(j searchJob) {
			defer subWg.Done()
			items := fetchGitHubSearch(j.query, j.sortBy)
			for _, item := range items {
				if item.FullName == "" || item.Archived {
					continue
				}
				mu.Lock()
				if seenRepos[item.FullName] {
					mu.Unlock()
					continue
				}
				seenRepos[item.FullName] = true
				mu.Unlock()

				title := cleanHTMLAndMarkdown(item.FullName, false)
				desc := cleanHTMLAndMarkdown(item.Description, false)
				if desc == "" {
					desc = "[Brak opisu repozytorium]"
				}
				lang := item.Language
				if lang == "" {
					lang = "N/A"
				}
				licenseStr := "N/A"
				if item.License != nil && item.License.Name != "" {
					licenseStr = item.License.Name
				}
				topicsStr := ""
				if len(item.Topics) > 0 {
					maxTopics := 5
					if len(item.Topics) < maxTopics {
						maxTopics = len(item.Topics)
					}
					topicsStr = " | Tagi: " + strings.Join(item.Topics[:maxTopics], ", ")
				}
				updatedStr := ""
				if item.UpdatedAt != "" {
					if t, err := time.Parse(time.RFC3339, item.UpdatedAt); err == nil {
						updatedStr = fmt.Sprintf(" | Aktl: %s", t.Format("2006-01-02"))
					}
				}
				body := fmt.Sprintf("Stars: %d | Forks: %d | Lang: %s | Lic: %s%s%s | %s",
					item.Stars, item.Forks, lang, licenseStr, topicsStr, updatedStr, desc)
				score := scoreGitHubItem(rawQ, item) + (priority * 4)

				mu.Lock()
				allResults = append(allResults, BoltResult{"GITHUB REPO", title, body, item.HtmlUrl, "", score})
				mu.Unlock()
			}
		}(job)
	}

	subWg.Wait()

	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	limit := len(allResults)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		ch <- allResults[i]
	}
}

func fetchWiki(rawQ string, ch chan<- BoltResult, wg *sync.WaitGroup, priority int) {
	defer wg.Done()
	apiQ := buildAPIQuery(rawQ)
	u := b64("aHR0cHM6Ly9lbi53aWtpcGVkaWEub3JnL3cvYXBpLnBocD9hY3Rpb249cXVlcnkmbGlzdD1zZWFyY2gmc3JsaW1pdD01JmZvcm1hdD1qc29uJnNyc2VhcmNoPQ==") + url.QueryEscape(apiQ)
	resp, err := getRaw(u)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()

	var res struct {
		Query struct {
			Search []struct {
				Title string `json:"title"`
			} `json:"search"`
		} `json:"query"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
		var wikiWg sync.WaitGroup
		for _, item := range res.Query.Search {
			wikiWg.Add(1)
			go func(titleStr string) {
				defer wikiWg.Done()
				sumU := b64("aHR0cHM6Ly9lbi53aWtpcGVkaWEub3JnL2FwaS9yZXN0X3YxL3BhZ2Uvc3VtbWFyeS8=") + url.PathEscape(titleStr)
				sumResp, err := getRaw(sumU)
				if err == nil && sumResp.StatusCode == 200 {
					var finalRes struct {
						Title   string
						Extract string
					}
					json.NewDecoder(sumResp.Body).Decode(&finalRes)
					if finalRes.Extract != "" {
						cleanExtract := cleanHTMLAndMarkdown(finalRes.Extract, false)
						linkStr := b64("aHR0cHM6Ly9lbi53aWtpcGVkaWEub3JnL3dpa2kv") + strings.ReplaceAll(titleStr, " ", "_")
						score := calculateScore(rawQ, finalRes.Title, cleanExtract, "WIKIPEDIA", priority)
						ch <- BoltResult{"WIKIPEDIA", finalRes.Title, cleanExtract, linkStr, "", score}
					}
					sumResp.Body.Close()
				}
			}(item.Title)
		}
		wikiWg.Wait()
	}
}

func fetchHackerNews(rawQ string, ch chan<- BoltResult, wg *sync.WaitGroup, priority int) {
	defer wg.Done()
	apiQ := buildAPIQuery(rawQ)

	u := b64("aHR0cHM6Ly9obi5hbGdvbGlhLmNvbS9hcGkvdjEvc2VhcmNoP3RhZ3M9c3RvcnkmaGl0c1BlclBhZ2U9NSZxdWVyeT0=") + url.QueryEscape(apiQ)
	resp, err := getRaw(u)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()

	var res struct {
		Hits []struct {
			Title    string `json:"title"`
			Story    string `json:"story_text"`
			ObjectID string `json:"objectID"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
		for _, hit := range res.Hits {
			title := cleanHTMLAndMarkdown(hit.Title, false)
			if title == "" {
				continue
			}
			code, textWithoutCode := extractCode(hit.Story)
			text := cleanHTMLAndMarkdown(textWithoutCode, false)
			if text == "" {
				text = "[Dyskusja w linku]"
			}
			fullURL := b64("aHR0cHM6Ly9uZXdzLnljb21iaW5hdG9yLmNvbS9pdGVtP2lkPQ==") + hit.ObjectID
			score := calculateScore(rawQ, title, text, "HACKER NEWS", priority)
			ch <- BoltResult{"HACKER NEWS", title, text, fullURL, code, score}
		}
	}
}

func fetchStackOverflow(rawQ string, ch chan<- BoltResult, wg *sync.WaitGroup, priority int) {
	defer wg.Done()
	apiQ := buildAPIQuery(rawQ)
	u := b64("aHR0cHM6Ly9hcGkuc3RhY2tleGNoYW5nZS5jb20vMi4zL3NlYXJjaC9hZHZhbmNlZD9vcmRlcj1kZXNjJnNvcnQ9dm90ZXMmYWNjZXB0ZWQ9VHJ1ZSZzaXRlPXN0YWNrb3ZlcmZsb3cmcT0=") + url.QueryEscape(apiQ)
	resp, err := getRaw(u)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()

	var res struct {
		Items []struct {
			Title      string `json:"title"`
			Excerpt    string `json:"excerpt"`
			QuestionID int    `json:"question_id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
		for _, item := range res.Items {
			if item.Title == "" {
				continue
			}
			title := cleanHTMLAndMarkdown(item.Title, false)
			code, textWithoutCode := extractCode(item.Excerpt)
			excerpt := cleanHTMLAndMarkdown(textWithoutCode, false)
			fullURL := fmt.Sprintf(b64("aHR0cHM6Ly9zdGFja292ZXJmbG93LmNvbS9xLyVk"), item.QuestionID)
			score := calculateScore(rawQ, title, excerpt, "STACK OVERFLOW", priority)
			ch <- BoltResult{"STACK OVERFLOW", title, excerpt, fullURL, code, score}
		}
	}
}

func fetchDevTo(rawQ string, ch chan<- BoltResult, wg *sync.WaitGroup, priority int) {
	defer wg.Done()
	apiQ := buildAPIQuery(rawQ)
	u := b64("aHR0cHM6Ly9kZXYudG8vYXBpL2FydGljbGVzP3Blcl9wYWdlPTUmc2VhcmNoPQ==") + url.QueryEscape(apiQ)
	resp, err := getRaw(u)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()

	var res []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
		for _, item := range res {
			if item.Title == "" {
				continue
			}
			title := cleanHTMLAndMarkdown(item.Title, false)
			code, textWithoutCode := extractCode(item.Description)
			desc := cleanHTMLAndMarkdown(textWithoutCode, false)
			score := calculateScore(rawQ, title, desc, "DEV.TO", priority)
			ch <- BoltResult{"DEV.TO", title, desc, item.URL, code, score}
		}
	}
}

func fetchAIResearch(q string) {
	fmt.Println("\033[36mZapytanie do serwerów AI...\033[0m")

	prompt := `You are an elite Senior Software Engineer, Technical Architect, and Principal Security Researcher.
Your goal is to provide the most precise, technically accurate, and comprehensive answer possible in Markdown format.

Structure your response exactly as follows:
1. ### EXECUTIVE SUMMARY: A direct, one-sentence answer to the query.
2. ### DEEP DIVE: A detailed technical explanation of the architecture, protocols, or logic involved.
3. ### PRODUCTION CODE: Provide modern, production-ready code snippets with detailed comments. Use best practices.
4. ### PITFALLS & SECURITY: Explicitly list common mistakes, security vulnerabilities, or performance bottlenecks.
5. ### MODERN ECOSYSTEM: Mention relevant tools, libraries, or frameworks that are industry standard in 2024/2025.

Rules:
- NO generic introductions like "As an AI..." or "Here is the information...".
- Be concise but extremely technical. Assume the user is a Lead Developer.
- If the question is non-technical, provide the most relevant historical or factual data with similar precision.

Question: ` + q
	plPayload := fmt.Sprintf(`{"messages":[{"role":"user","content":%q}],"model":"openai","json":false}`, prompt)
	plReq, _ := http.NewRequest("POST", "https://text.pollinations.ai/", strings.NewReader(plPayload))
	plReq.Header.Set("Content-Type", "application/json")
	plResp, err := httpClient.Do(plReq)
	if err == nil && plResp.StatusCode == 200 {
		body, readErr := io.ReadAll(plResp.Body)
		plResp.Body.Close()
		if readErr == nil && len(body) > 20 {
			formattedText := formatForTerminal(string(body))
			fmt.Printf("\n%s SYNTEZA AI %s %s[POLLINATIONS AI]%s %s%s%s\n\n", ansiBlueLine, ansiReset, ansiMagenta, ansiReset, ansiBold, q, ansiReset)
			fmt.Println(formattedText)
			fmt.Println("\n" + strings.Repeat("-", 80))
			return
		}
	}
	if plResp != nil {
		plResp.Body.Close()
	}

	fmt.Println("\033[33mPollinations niedostępne, próbuję DuckDuckGo AI...\033[0m")
	vqd := getVQD()
	if vqd != "" {
		ddgPayload := fmt.Sprintf(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":%q}]}`, prompt)
		ddgReq, _ := http.NewRequest("POST", "https://duckduckgo.com/duckchat/v1/chat", strings.NewReader(ddgPayload))
		ddgReq.Header.Set("Content-Type", "application/json")
		ddgReq.Header.Set("x-vqd-4", vqd)
		ddgReq.Header.Set("User-Agent", "BoltZenithApp/62.0")
		ddgResp, err := httpClient.Do(ddgReq)
		if err == nil && ddgResp.StatusCode == 200 {
			scanner := bufio.NewScanner(ddgResp.Body)
			var sb strings.Builder
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					if data == "[DONE]" {
						break
					}
					var chunk struct {
						Message string `json:"message"`
					}
					json.Unmarshal([]byte(data), &chunk)
					sb.WriteString(chunk.Message)
				}
			}
			ddgResp.Body.Close()
			if sb.Len() > 10 {
				formattedText := formatForTerminal(sb.String())
				fmt.Printf("\n%s SYNTEZA AI %s %s[DUCKDUCKGO AI]%s %s%s%s\n\n", ansiBlueLine, ansiReset, ansiMagenta, ansiReset, ansiBold, q, ansiReset)
				fmt.Println(formattedText)
				fmt.Println("\n" + strings.Repeat("-", 80))
				return
			}
		}
		if ddgResp != nil {
			ddgResp.Body.Close()
		}
	}

	fmt.Println("\033[31m[!] All AI providers unavailable.\033[0m")
}

func optimizeQueryWithAI(query string) string {
	prompt := `You are an elite technical search query optimizer.
Your goal is to transform the user's input into the most effective search query for technical engines (GitHub, StackOverflow).

Rules:
1. Identify the CORE technical intent.
2. DISCARD all conversational filler, greetings, and polite phrases.
3. EXTRACT absolute critical keywords: Tech stack names, error codes, specific functions/methods, and architectural patterns.
4. Translate all concepts to standard English industry terminology.
5. If searching for a person or specific tool, include the exact name and category.
6. DO NOT add words that are not present in the original query unless essential for context.
7. Max 4-6 words. Output ONLY keywords separated by spaces. No punctuation.

Input: ` + query

	u := b64("aHR0cHM6Ly90ZXh0LnBvbGxpbmF0aW9ucy5haS8=") + url.PathEscape(prompt)
	resp, err := getRaw(u)
	if err == nil && resp.StatusCode == 200 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr == nil && len(bodyBytes) > 0 {
			res := cleanHTMLAndMarkdown(string(bodyBytes), false)
			res = rePunct.ReplaceAllString(res, " ")
			res = strings.TrimSpace(res)
			if len(res) > 0 && len(res) < 200 && !strings.Contains(strings.ToLower(res), "sorry") {
				return res
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	ddgPayload := fmt.Sprintf(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":%q}]}`, prompt)
	ddgReq, err := http.NewRequest("POST", "https://duckduckgo.com/aichat/v1/chat", strings.NewReader(ddgPayload))
	if err == nil {
		ddgReq.Header.Set("Content-Type", "application/json")
		ddgReq.Header.Set("User-Agent", "BoltZenithApp/62.0")
		ddgReq.Header.Set("x-vqd-accept", "1")
		ddgResp, err := httpClient.Do(ddgReq)
		if err == nil && ddgResp.StatusCode == 200 {
			ddgBody, readErr := io.ReadAll(ddgResp.Body)
			ddgResp.Body.Close()
			if readErr == nil && len(ddgBody) > 0 {
				res := cleanHTMLAndMarkdown(string(ddgBody), false)
				res = rePunct.ReplaceAllString(res, " ")
				res = strings.TrimSpace(res)
				if len(res) > 0 && len(res) < 200 && !strings.Contains(strings.ToLower(res), "sorry") {
					return res
				}
			}
		} else if ddgResp != nil {
			ddgResp.Body.Close()
		}
	}

	return query
}

func preSearchBrain(query string) BrainResult {
	prompt := `You are the ultimate technical tokenization engine. Analyze the technical search query and return ONLY a valid JSON object.

Input: "` + query + `"

Requirements:
1. "english_q": Translate/Refine to professional technical English.
2. "optimized_q": 3-6 words representing ONLY the core technical intent provided in the input.
3. "primary_tokens": The absolute core technology/library/language mentioned in the input.
4. "action_tokens": The specific intent or error mentioned in the input.
5. "context_tokens": Constraints mentioned like "arm64", "v1.23", "production", "linux".
6. "exclude_tokens": Technical terms the user explicitly wants to avoid.
7. "targets": Map of sources (GITHUB, STACK OVERFLOW, HACKER NEWS, DEV.TO, WIKIPEDIA) to priority (0-100).
8. "insight": One sentence of high-level expert advice.
9. CRITICAL: DO NOT add extra keywords that were not in the user query (e.g. if I type 'csharp' do not add 'async' unless I asked for it).

JSON Format:
{"english_q": "...", "optimized_q": "...", "primary_tokens": "...", "action_tokens": "...", "context_tokens": "...", "exclude_tokens": "...", "targets": {"GITHUB": 80, ...}, "insight": "..."}
JSON Output: `

	parseBrain := func(body string) BrainResult {
		body = cleanHTMLAndMarkdown(body, true)
		start := strings.Index(body, "{")
		end := strings.LastIndex(body, "}")
		if start != -1 && end != -1 && end > start {
			body = body[start : end+1]
		}
		var res BrainResult
		if err := json.Unmarshal([]byte(body), &res); err == nil && res.EnglishQ != "" {
			return res
		}
		return BrainResult{}
	}

	aiResponse := fetchSimpleAI(prompt)
	if res := parseBrain(aiResponse); res.EnglishQ != "" {
		return res
	}

	return BrainResult{
		EnglishQ:   query,
		OptimizedQ: query,
		Targets:    map[string]int{"STACK OVERFLOW": 80, "GITHUB": 50},
		Insight:    "Using standard heuristics (AI Brain timeout).",
	}
}

func getAITargetScores(query string) map[string]int {
	prompt := `You are an advanced technical AI router. Your goal is to select the best search engines for a given query.
Analysis Rules:
- GITHUB: High score for library searches, source code, repositories, or "how to implement X".
- STACK OVERFLOW: High score for errors, bugs, "how to do X", or specific syntax questions.
- HACKER NEWS: High score for "why is X popular", "opinions on X", "latest news in tech", or famous figures in CS/Hacking.
- DEV.TO: High score for tutorials, blog posts, comparisons, or beginner guides.
- WIKIPEDIA: High score for historical figures, general definitions, non-technical history (e.g., "who was X"), or broad concepts.

Query: ` + query + `
Output valid JSON map only. Example: {"GITHUB": 10, "STACK OVERFLOW": 80, "WIKIPEDIA": 5}
JSON Output: `

	parseAIResponse := func(body string) map[string]int {
		body = cleanHTMLAndMarkdown(body, true)
		start := strings.Index(body, "{")
		end := strings.LastIndex(body, "}")
		if start != -1 && end != -1 && end > start {
			body = body[start : end+1]
		}
		var parsed map[string]int
		if err := json.Unmarshal([]byte(body), &parsed); err == nil {
			return parsed
		}
		return nil
	}

	u := b64("aHR0cHM6Ly90ZXh0LnBvbGxpbmF0aW9ucy5haS8=") + url.PathEscape(prompt)
	resp, err := getRaw(u)
	if err == nil && resp.StatusCode == 200 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr == nil && len(bodyBytes) > 0 {
			if parsed := parseAIResponse(string(bodyBytes)); parsed != nil {
				return parsed
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	ddgPayload := fmt.Sprintf(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":%q}]}`, prompt)
	ddgReq, err := http.NewRequest("POST", "https://duckduckgo.com/aichat/v1/chat", strings.NewReader(ddgPayload))
	if err == nil {
		ddgReq.Header.Set("Content-Type", "application/json")
		ddgReq.Header.Set("User-Agent", "BoltZenithApp/62.0")
		ddgReq.Header.Set("x-vqd-accept", "1")
		ddgResp, err := httpClient.Do(ddgReq)
		if err == nil && ddgResp.StatusCode == 200 {
			ddgBody, readErr := io.ReadAll(ddgResp.Body)
			ddgResp.Body.Close()
			if readErr == nil && len(ddgBody) > 0 {
				if parsed := parseAIResponse(string(ddgBody)); parsed != nil {
					return parsed
				}
			}
		} else if ddgResp != nil {
			ddgResp.Body.Close()
		}
	}

	return map[string]int{
		"GITHUB":         0,
		"STACK OVERFLOW": 0,
		"HACKER NEWS":    0,
		"DEV.TO":         0,
		"WIKIPEDIA":      100,
	}
}

func classifyVideoCategory(title string, desc string) string {
	tl := strings.ToLower(title)
	dl := strings.ToLower(desc)
	combined := tl + " " + dl

	tutorialSignals := []string{
		"tutorial", "how to", "step by step", "beginner", "learn", "guide", "getting started", "from scratch", "walkthrough", "explained", "for beginners", "introduction to", "basics", "fundamentals", "course", "lesson",
		"poradnik", "kurs", "lekcja", "przewodnik", "wstęp", "wprowadzenie", "krok po kroku", "od zera", "podstawy", "nauka", "zrozumieć", "opanować", "manual", "handbook", "primer", "bootcamp", "training", "exercise",
		"practice", "workshop", "lab", "assignment", "homework", "example", "sample", "demo", "quickstart", "cheat sheet", "cheatsheet", "roadmap", "pathway", "curriculum", "syllabus", "module", "chapter", "section",
		"part 1", "part 2", "series", "playlist", "complete guide", "ultimate guide", "comprehensive guide", "mastering", "deep dive", "internals", "architecture", "design patterns", "best practices", "clean code",
		"solid", "tdd", "testing", "debugging", "deployment", "hosting", "production", "real world", "application", "project", "build a", "create a", "make a", "developing", "coding", "programming", "scripting",
		"automation", "integration", "api", "database", "frontend", "backend", "fullstack", "devops", "cloud", "serverless", "microservices", "docker", "kubernetes", "git", "github", "setup", "install", "config",
		"refactoring", "optimization", "performance", "security", "exploit", "penetration", "hacker", "white hat", "black hat", "defense", "mitigation", "incident", "response", "forensics", "malware", "analysis", "reverse engineering",
		"cryptography", "blockchain", "smart contract", "ethereum", "bitcoin", "nft", "dao", "web3", "ai", "machine learning", "deep learning", "neural network", "transformer", "bert", "gpt", "model", "inference", "training",
		"dataset", "data science", "analytics", "visualization", "bi", "data warehouse", "data lake", "data pipeline", "etl", "scraping", "crawler", "browser", "engine", "parser", "compiler", "interpreter", "runtime",
	}
	crashCourseSignals := []string{
		"crash course", "full course", "complete course", "in one video", "all you need", "masterclass", "bootcamp", "zero to hero", "complete guide",
		"wszystko w jednym", "cały kurs", "kompletny kurs", "szybki kurs", "intensywny kurs", "skondensowana wiedza", "mastery", "full tutorial", "marathon", "mega course", "giant course", "long video", "everything you need",
		"1 hour", "2 hours", "3 hours", "5 hours", "10 hours", "full project", "end to end", "e2e", "comprehensive", "all-in-one", "ultimate", "absolute", "complete walkthrough", "full implementation", "build from scratch",
		"from start to finish", "from A to Z", "beginner to pro", "junior to senior", "zero to mastery", "the complete", "the ultimate", "the masterclass", "full series", "box set", "compilation", "collection", "bundle",
		"speedrun", "fast track", "accelerated", "intensive", "power course", "essential", "core", "foundation", "pro guide", "expert guide", "senior guide", "architect guide", "lead guide", "full stack guide", "backend guide",
		"frontend guide", "mobile guide", "game dev guide", "ai guide", "data science guide", "devops guide", "security guide", "cloud guide", "blockchain guide", "web3 guide", "soft skills guide", "career guide", "job guide",
		"the complete series", "the whole course", "the entire guide", "everything inklusiv", "all topics", "comprehensive training", "one stop shop", "no experience needed", "zero background", "start to finish guide", "comprehensive walk",
		"ultimate masterclass", "the final guide", "all in one place", "the only video", "the primary course", "the fundamental course", "the core guide", "the absolute guide", "the essential guide", "the professional guide",
		"the enterprise guide", "the scale guide", "the performance guide", "the optimization guide", "the security guide", "the architecture guide", "the internals guide", "the deep dive guide", "the low level guide",
	}
	comparisonSignals := []string{
		"vs", "versus", "compared", "comparison", "which is better", "differences", "pros and cons", "benchmark", "performance", "speed", "test",
		"porównanie", "kontra", "co lepsze", "różnice", "za i przeciw", "wady i zalety", "recenzja", "opinia", "ranking", "top 5", "top 10", "best of", "winner", "loser", "choice", "selection", "decision", "tradeoffs",
		"alternatives", "options", "competitors", "market share", "popularity", "trends", "stack", "tools", "frameworks", "libraries", "languages", "databases", "platforms", "services", "providers", "hosting", "cloud",
		"rust vs go", "react vs vue", "angular vs react", "next vs nuxt", "python vs r", "java vs c#", "sql vs nosql", "docker vs podman", "k8s vs swarm", "aws vs azure", "gcp vs aws", "terraform vs pulumi", "mac vs pc",
		"ios vs android", "native vs cross", "flutter vs rn", "swift vs kotlin", "php vs node", "fastapi vs flask", "django vs rails", "tailwind vs bootstrap", "css vs sass", "vite vs webpack", "npm vs yarn", "pnpm vs npm",
		"better than", "worse than", "faster than", "slower than", "easier than", "harder than", "cheaper than", "pricier than", "best framework", "best language", "best tool", "best library", "best database", "best cloud",
		"top choice", "number one", "the winner is", "who wins", "showdown", "face off", "battle", "clash", "fight", "competition", "rivalry", "matchup", "head to head", "split", "divide", "alternatives to", "replaces",
	}
	liveCodingSignals := []string{
		"live coding", "code along", "build with me", "let's build", "coding challenge", "project tutorial", "real world", "from zero",
		"kodowanie na żywo", "buduj ze mną", "projekt od zera", "wyzwanie", "praktyka", "realny projekt", "live stream", "recorded live", "unfiltered", "unscripted", "raw coding", "authentic", "problem solving", "live debug",
		"build together", "pair programming", "interactive", "qa session", "ask me anything", "ama", "live demo", "live workshop", "hands on", "practical coding", "watch me code", "code marathon", "coding session", "hackathon",
		"build a saas", "build a clone", "build from prototype", "from idea to production", "live refactor", "live migration", "live deploy", "live fix", "live audit", "live security", "live test", "live devops", "live cloud",
		"build in public", "indie hacker", "startup live", "mvp build", "rapid prototyping", "fast coding", "clean coding live", "senior coding live", "lead coding live", "expert coding live", "architect coding live",
		"raw stream", "uncut", "unmodified", "unedited", "live problem", "solving live", "coding in real time", "real time dev", "dev diary", "dev log", "coding vlog", "stream highlight", "stream session", "code and chill",
		"lofi hip hop", "background music", "coworking", "study with me", "focus session", "deep work", "deep focus", "coding music", "dark mode coding", "night coding", "chill coding", "relaxing coding", "productive coding",
		"focusing", "quiet time", "pomodoro", "study session", "work with me", "day in the life", "morning routine", "night routine", "home office", "desk setup", "minimalist", "gaming setup", "setup tour", "pc build",
	}
	talkSignals := []string{
		"conference", "keynote", "talk", "presentation", "meetup", "summit", "panel", "fireside", "interview", "discussion", "debate", "speech", "lecture",
		"konferencja", "prezentacja", "spotkanie", "wywiad", "dyskusja", "panel", "przemówienie", "wykład", "seminarium", "webinar", "podcast", "event", "gathering", "forum", "symposium", "convention", "expo", "trade show",
		"devconf", "con", "summit 2024", "summit 2025", "google io", "apple wwdc", "microsoft build", "aws re:invent", "kubecon", "dockercon", "rustconf", "gophercon", "pycon", "jsconf", "reactconf", "vueconf", "angularconf",
		"defcon", "blackhat", "security conference", "ai summit", "ml conference", "data summit", "cloud summit", "devops days", "agile tour", "scrum gathering", "tech talk", "tech presentation", "engineering talk", "architect talk",
		"fosdem", "linuxcon", "oscon", "google cloud next", "oracle code", "ibm think", "re:invent", "ignite", "build", "wwdc", "io", "fb developer", "f8", "collision", "web summit", "slush", "sxsw", "ces", "computex", "ifa",
		"e3", "gamescom", "gdc", "siggraph", "iclr", "nips", "neurips", "cvpr", "eccv", "iccv", "aaai", "ijcai", "emnlp", "acl", "naacl", "icml", "kdd", "sigir", "www", "infocom", "sigcomm", "mobicom", "sensys", "asplos",
	}
	officialSignals := []string{
		"official", "documentation", "release", "announcement", "changelog", "what's new", "update", "roadmap", "status", "report", "brief", "memo", "guide",
		"oficjalna", "dokumentacja", "wydanie", "ogłoszenie", "zmiany", "nowości", "aktualizacja", "plany", "raport", "informacja", "przewodnik", "instrukcja", "oficjalny kanał", "official channel", "official video", "by the founders",
		"core team", "maintainers", "community update", "foundation update", "standard", "specification", "rfc", "iso", "ieee", "w3c", "ecma", "proposal", "draft", "v1.0", "v2.0", "v3.0", "major release", "minor release",
		"patch release", "official demo", "official teaser", "official trailer", "official launch", "official event", "official partnership", "official integration", "official support", "official statement", "official news",
		"presse", "media", "pr", "prm", "briefing", "official blog", "official tweet", "official post", "official repo", "original source", "authorized", "certified", "validated", "verified source", "legit", "authentic source",
		"official app", "official site", "official page", "landingspage", "product page", "features list", "pricing", "plans", "enterprise edition", "community edition", "professional edition", "ultimate edition", "free tier",
		"standardization", "blueprint", "canonical source", "official documentation", "main docs", "official faq", "official wiki", "official forums", "official support", "official help", "official contact", "official team",
		"official partners", "official affiliates", "official distributors", "official vendors", "official store", "official merchandise", "official podcast", "official newsletter", "official github", "official repo",
	}
	reviewSignals := []string{
		"review", "honest review", "opinion", "worth it", "should you", "my experience", "my thoughts", "evaluation", "score", "grade", "rating", "verdict",
		"recenzja", "szczera opinia", "warto", "czy warto", "moje doświadczenie", "moja opinia", "testujemy", "sprawdzamy", "używamy", "analiza", "werdykt", "ocena", "podsumowanie", "final thoughts", "long term review",
		"after 1 year", "after 6 months", "after 1 month", "daily driver", "switched to", "i quit", "why i left", "why i moved", "why i chose", "why i use", "comparison review", "hands on review", "unboxing", "first look",
		"initial impressions", "deep review", "expert review", "professional review", "user review", "community review", "honest take", "unbiased review", "critical review", "detailed review", "quick review", "summary review",
		"the truth", "bad parts", "good parts", "annoying parts", "limitations", "weaknesses", "strengths", "capabilities", "features review", "performance review", "ux review", "ui review", "dx review", "developer review",
		"real review", "neutral review", "objective review", "subjective review", "personal review", "final verdict", "conclusion", "should i buy", "should i use", "is it dead", "is it obsolete", "is it future proof",
		"long term evaluation", "retrospective review", "year in review", "state of the union", "comprehensive analysis", "detailed evaluation", "pros and cons analysis", "technical comparison", "feature by feature",
		"benchmark results", "performance labs", "test results", "real world testing", "case study", "user experience report", "developer experience survey", "industry review", "market analysis", "expert opinion",
	}
	advancedSignals := []string{
		"advanced", "senior", "architecture", "deep dive", "internals", "expert", "mastery", "in depth", "low level", "performance optimization", "security audit", "under the hood", "production ready",
		"zaawansowane", "ekspert", "architektura", "szczegółowo", "wnętrze", "niskopoziomowy", "optymalizacja", "produkcyjne", "profesjonalne", "senior dev", "lead dev", "staff dev", "principal dev", "cto level", "engineering",
		"distributed systems", "concurrency", "parallelism", "scalability", "resilience", "high availability", "fault tolerance", "observability", "metrics", "tracing", "logging", "memory management", "garbage collection",
		"jit compiler", "interpreter", "runtime internals", "kernel dev", "drivers", "embedded systems", "iot architecture", "cloud native", "kubernetes internals", "docker internals", "database internals", "query optimization",
		"indexing strategies", "security exploitation", "reverse engineering", "binary analysis", "cryptography", "math for devs", "algorithms analysis", "big o", "data structures dive", "designing data intensive", "clean architecture",
		"microservices architecture", "event driven", "cqrs", "event sourcing", "domain driven design", "ddd", "hexagonal architecture", "onion architecture", "modular monolith", "distributed database", "paxos", "raft", "consensus",
		"byzantine fault tolerance", "bft", "sharding", "partitioning", "replication", "consistency", "availability", "partition tolerance", "cap theorem", "acid vs base", "stateless architecture", "edge computing", "serverless internals",
		"assembly language", "machine code", "binary format", "executable internals", "linking and loading", "memory layout", "calling conventions", "stack frames", "heap allocation algorithms", "cache locality", "simd",
		"vectorization", "parallel processing", "multithreading models", "lock free programming", "atomic operations", "memory fences", "wait free data structures", "distributed consensus", "paxos", "raft", "zab", "viewstamped",
	}

	advScore := 0
	for _, s := range advancedSignals {
		if strings.Contains(combined, s) {
			advScore++
		}
	}
	if advScore >= 1 {
		return "ADVANCED/SENIOR"
	}

	crashScore := 0
	for _, s := range crashCourseSignals {
		if strings.Contains(combined, s) {
			crashScore++
		}
	}
	if crashScore > 0 {
		return "CRASH COURSE"
	}

	liveCodingScore := 0
	for _, s := range liveCodingSignals {
		if strings.Contains(combined, s) {
			liveCodingScore++
		}
	}
	if liveCodingScore > 0 {
		return "LIVE CODING"
	}

	tutScore := 0
	for _, s := range tutorialSignals {
		if strings.Contains(combined, s) {
			tutScore++
		}
	}
	if tutScore >= 2 {
		return "TUTORIAL"
	}

	for _, s := range comparisonSignals {
		if strings.Contains(combined, s) {
			return "COMPARISON"
		}
	}
	for _, s := range talkSignals {
		if strings.Contains(combined, s) {
			return "TALK"
		}
	}
	for _, s := range officialSignals {
		if strings.Contains(combined, s) {
			return "OFFICIAL"
		}
	}
	for _, s := range reviewSignals {
		if strings.Contains(combined, s) {
			return "REVIEW"
		}
	}
	if tutScore > 0 {
		return "TUTORIAL"
	}

	return "VIDEO"
}

func scoreYouTubeItem(rawQ string, title string, desc string, author string, viewCount int64, lengthSecs int) int {
	score := 50
	qLower := strings.ToLower(rawQ)
	qClean := rePunct.ReplaceAllString(qLower, " ")
	words := strings.Fields(qClean)
	tLower := strings.ToLower(title)
	dLower := strings.ToLower(desc)
	tClean := rePunct.ReplaceAllString(tLower, " ")

	titleMatches := 0
	descMatches := 0
	validWords := 0

	for _, w := range words {
		if len(w) < 3 {
			continue
		}
		validWords++

		if strings.Contains(tClean, w) {
			titleMatches++
			if massiveTechDB[w] {
				score += 250
			} else {
				score += 150
			}
		}
		if strings.Contains(dLower, w) {
			descMatches++
			score += 60
		}
		if syns, ok := synonymDB[w]; ok {
			for _, syn := range syns {
				if strings.Contains(tClean, syn) {
					score += 80
					break
				}
			}
		}
		for key, syns := range synonymDB {
			if key == w {
				continue
			}
			for _, syn := range syns {
				if syn == w {
					if strings.Contains(tClean, key) {
						score += 60
					}
					break
				}
			}
		}
	}

	for idx := 0; idx < len(words)-1; idx++ {
		if len(words[idx]) < 3 || len(words[idx+1]) < 3 {
			continue
		}
		pair := words[idx] + " " + words[idx+1]
		if strings.Contains(tClean, pair) {
			score += 300
		}
		if strings.Contains(dLower, pair) {
			score += 120
		}
	}

	if validWords > 0 {
		ratio := float64(titleMatches) / float64(validWords)
		if ratio == 1.0 {
			score += 1500
		} else if ratio > 0.7 {
			score += 800
		} else if ratio > 0.5 {
			score += 400
		}
		if titleMatches == 0 && descMatches == 0 {
			score -= 1000
		} else if titleMatches == 0 {
			score -= 300
		}
	}

	if strings.Contains(tLower, qLower) {
		score += 2000
	}

	category := classifyVideoCategory(title, desc)
	isHowTo := strings.Contains(qLower, "how") || strings.Contains(qLower, "tutorial") || strings.Contains(qLower, "learn") || strings.Contains(qLower, "guide")
	isCompare := strings.Contains(qLower, "vs") || strings.Contains(qLower, "best") || strings.Contains(qLower, "compare")

	switch category {
	case "TUTORIAL":
		score += 200
		if isHowTo {
			score += 300
		}
	case "CRASH COURSE":
		score += 300
		if isHowTo {
			score += 400
		}
	case "LIVE CODING":
		score += 250
		if isHowTo {
			score += 200
		}
	case "COMPARISON":
		score += 150
		if isCompare {
			score += 400
		}
	case "TALK":
		score += 100
	case "OFFICIAL":
		score += 200
	case "REVIEW":
		score += 100
		if isCompare {
			score += 200
		}
	case "ADVANCED/SENIOR":
		score += 600
		if isHowTo {
			score += 400
		}
	}

	knownChannels := map[string]int{
		"traversy media": 400, "fireship": 500, "the coding train": 350,
		"web dev simplified": 400, "techworld with nana": 350, "freecodecamp": 500,
		"programming with mosh": 400, "academind": 350, "the net ninja": 400,
		"sentdex": 350, "corey schafer": 400, "tech with tim": 300,
		"computerphile": 350, "clément mihailescu": 300, "neetcode": 400,
		"ben awad": 300, "theo": 350, "primeagen": 350, "jack herrington": 300,
		"coding with john": 300, "bro code": 300, "caleb curry": 250,
		"hussein nasser": 350, "arjan codes": 300, "anthonygg": 250,
		"dreams of code": 300, "typecraft": 250, "devops toolkit": 300,
		"coding addict": 300, "online tutorials": 250, "kevin powell": 300,
		"codevolution": 300, "developedbyed": 300, "james q quick": 250,
		"joshua morony": 250, "leigieber": 250, "william lin": 350,
		"erik dot dev": 250, "coder coder": 250, "designcourse": 300,
		"flux academy": 300, "paddy gulas": 250, "gary explains": 300,
		"the nexus": 250, "low level learning": 350, "hackersploit": 350,
		"networkchuck": 400, "david bombal": 400, "null byte": 300,
		"infosec pat": 300, "the cyber mentor": 400, "ippsec": 400,
		"liveoverflow": 400, "stok": 300, "nahamsec": 350,
		"john hammond": 450, "computer science": 300, "mit opencourseware": 500,
		"stanford": 450, "harvard cs50": 500, "edx": 350,
		"coursera": 350, "udemy": 300, "pluralsight": 350,
		"eggheadio": 350, "level up tutorials": 300, "wes bos": 350,
		"scrimba": 300, "frontend masters": 450, "dotconferences": 400,
		"ndc conferences": 400, "gotoconferences": 400, "oreilly": 350,
		"google developers": 450, "android developers": 400, "apple developer": 400,
		"microsoft developer": 400, "aws wales": 350, "hashicorp": 400,
		"docker": 400, "kubernetes": 400, "cncf": 450,
		"linux foundation": 450, "red hat": 400, "canonical": 350,
		"mongodb": 350, "elastic": 350, "redis": 350,
		"confluent": 350, "databricks": 350, "snowflake": 350,
		"cloudflare": 400, "digitalocean": 350, "linode": 300,
	}
	authorLower := strings.ToLower(author)
	for channel, bonus := range knownChannels {
		if strings.Contains(authorLower, channel) {
			score += bonus
			break
		}
	}

	if viewCount > 5000000 {
		score += 600
	} else if viewCount > 1000000 {
		score += 500
	} else if viewCount > 500000 {
		score += 400
	} else if viewCount > 100000 {
		score += 300
	} else if viewCount > 50000 {
		score += 200
	} else if viewCount > 10000 {
		score += 120
	} else if viewCount > 1000 {
		score += 50
	} else if viewCount < 100 {
		score -= 200
	}

	mins := lengthSecs / 60
	if isHowTo || category == "TUTORIAL" || category == "CRASH COURSE" {
		if mins >= 10 && mins <= 60 {
			score += 200
		} else if mins >= 5 && mins <= 120 {
			score += 100
		} else if mins < 2 {
			score -= 300
		}
	} else {
		if mins >= 5 && mins <= 30 {
			score += 150
		} else if mins >= 3 && mins <= 45 {
			score += 80
		} else if mins < 1 {
			score -= 200
		}
	}

	clickbaitSignals := []string{
		"you won't believe", "shocking", "insane", "mind blowing", "#shorts", "tiktok", "reaction", "prank", "gone wrong", "gone sexual", "emotional", "must watch", "unbelievable",
		"niesamowite", "nie uwierzysz", "szok", "szokujące", "petarda", "masakra", "reakcja", "żart", "prank", "śmieszne", "fail", "epic fail", "clickbait", "thumbnail", "giveaway",
		"free money", "make money", "passive income", "rich quickly", "scam", "shilling", "pump and dump", "crypto moon", "100x gem", "don't miss", "last chance", "hurry up", "limited time",
		"revealed", "exposed", "truth about", "hidden secret", "secret hack", "you need to see this", "the wait is over", "finally happened", "it's over", "i'm sorry", "we need to talk",
		"why i quit", "i'm leaving", "my biggest mistake", "don't do this", "stop doing this", "never do this", "the dark side", "the ugly truth", "nightmare", "horror", "scary",
		"ghost", "mystery", "solved", "leak", "rumor", "confirmed", "official trailer", "teaser", "leak", "stolen", "hacked", "breach", "warning", "danger", "deadly", "fatal", "dangerous",
		"extreme", "wild", "crazy", "unhinged", "epic", "legendary", "historic", "huge", "massive", "giant", "enormous", "incredible", "omg", "wow", "lol", "lmao", "rofl", "xd", "f", "rip", "rip coding", "dead",
		"killer", "savior", "hero", "villain", "monster", "beast", "god", "king", "queen", "lord", "master", "slave", "war", "battle", "clash", "fight", "struggle", "success", "failure", "win", "loss", "victory", "defeat",
	}
	for _, cb := range clickbaitSignals {
		if strings.Contains(tLower, cb) {
			score -= 500
			break
		}
	}

	return score
}

func fetchYouTube(rawQ string, ch chan<- BoltResult, wg *sync.WaitGroup) {
	defer wg.Done()
	apiQ := buildAPIQuery(rawQ)

	invidiousInstances := []string{
		"https://invidious.snopyta.org",
		"https://yewtu.be",
		"https://invidious.kavin.rocks",
		"https://vid.puffyan.us",
	}

	type ytItem struct {
		Title       string `json:"title"`
		VideoID     string `json:"videoId"`
		Author      string `json:"author"`
		Description string `json:"description"`
		ViewCount   int64  `json:"viewCount"`
		LengthSecs  int    `json:"lengthSeconds"`
		Published   int64  `json:"published"`
	}

	searchQueries := []string{
		apiQ + " tutorial",
		apiQ + " programming",
		apiQ,
	}

	seenVideos := make(map[string]bool)
	var allResults []ytItem

	for _, sq := range searchQueries {
		for _, instance := range invidiousInstances {
			u := instance + "/api/v1/search?q=" + url.QueryEscape(sq) + "&type=video&sort_by=relevance"
			resp, err := getRaw(u)
			if err != nil || resp.StatusCode != 200 {
				if resp != nil {
					resp.Body.Close()
				}
				continue
			}

			var items []ytItem
			if json.NewDecoder(resp.Body).Decode(&items) == nil && len(items) > 0 {
				resp.Body.Close()
				for _, item := range items {
					if !seenVideos[item.VideoID] && item.VideoID != "" {
						seenVideos[item.VideoID] = true
						allResults = append(allResults, item)
					}
				}
				break
			}
			resp.Body.Close()
		}
	}

	if len(allResults) == 0 {
		piped := "https://pipedapi.kavin.rocks/search?q=" + url.QueryEscape(apiQ) + "&filter=videos"
		resp, err := getRaw(piped)
		if err == nil && resp.StatusCode == 200 {
			var pipedRes struct {
				Items []struct {
					Title     string `json:"title"`
					URL       string `json:"url"`
					Duration  int    `json:"duration"`
					Views     int64  `json:"views"`
					Uploader  string `json:"uploaderName"`
					ShortDesc string `json:"shortDescription"`
				} `json:"items"`
			}
			if json.NewDecoder(resp.Body).Decode(&pipedRes) == nil {
				for _, pi := range pipedRes.Items {
					vidID := strings.TrimPrefix(pi.URL, "/watch?v=")
					if !seenVideos[vidID] && vidID != "" {
						seenVideos[vidID] = true
						allResults = append(allResults, ytItem{
							Title:       pi.Title,
							VideoID:     vidID,
							Author:      pi.Uploader,
							Description: pi.ShortDesc,
							ViewCount:   pi.Views,
							LengthSecs:  pi.Duration,
						})
					}
				}
			}
			resp.Body.Close()
		} else if resp != nil {
			resp.Body.Close()
		}
	}

	type scoredYT struct {
		item     ytItem
		score    int
		category string
	}

	var scored []scoredYT
	for _, item := range allResults {
		if item.Title == "" || item.VideoID == "" {
			continue
		}
		s := scoreYouTubeItem(rawQ, item.Title, item.Description, item.Author, item.ViewCount, item.LengthSecs)
		cat := classifyVideoCategory(item.Title, item.Description)
		scored = append(scored, scoredYT{item, s, cat})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	limit := len(scored)
	if limit > 10 {
		limit = 10
	}

	for i := 0; i < limit; i++ {
		item := scored[i].item
		sc := scored[i].score
		cat := scored[i].category

		durationStr := "N/A"
		if item.LengthSecs > 0 {
			mins := item.LengthSecs / 60
			secs := item.LengthSecs % 60
			if mins >= 60 {
				hours := mins / 60
				mins = mins % 60
				durationStr = fmt.Sprintf("%dh%02dm", hours, mins)
			} else {
				durationStr = fmt.Sprintf("%d:%02d", mins, secs)
			}
		}

		viewStr := "N/A"
		if item.ViewCount > 0 {
			if item.ViewCount >= 1000000 {
				viewStr = fmt.Sprintf("%.1fM", float64(item.ViewCount)/1000000)
			} else if item.ViewCount >= 1000 {
				viewStr = fmt.Sprintf("%.1fK", float64(item.ViewCount)/1000)
			} else {
				viewStr = fmt.Sprintf("%d", item.ViewCount)
			}
		}

		author := item.Author
		if author == "" {
			author = "Unknown"
		}

		desc := cleanHTMLAndMarkdown(item.Description, false)
		if len([]rune(desc)) > 200 {
			desc = string([]rune(desc)[:200]) + "..."
		}
		if desc == "" {
			desc = "[No description]"
		}

		catTag := ""
		switch cat {
		case "TUTORIAL":
			catTag = " | [TUTORIAL]"
		case "CRASH COURSE":
			catTag = " | [CRASH COURSE]"
		case "LIVE CODING":
			catTag = " | [LIVE CODING]"
		case "COMPARISON":
			catTag = " | [VS/COMPARISON]"
		case "TALK":
			catTag = " | [CONFERENCE TALK]"
		case "OFFICIAL":
			catTag = " | [OFFICIAL]"
		case "REVIEW":
			catTag = " | [REVIEW]"
		}

		body := fmt.Sprintf("%s | %s | %s%s | %s", author, durationStr, viewStr, catTag, desc)
		vidURL := "https://www.youtube.com/watch?v=" + item.VideoID

		ch <- BoltResult{"YOUTUBE", item.Title, body, vidURL, "", sc}
	}
}

func printResultLive(res BoltResult, index int) {
	printMu.Lock()
	defer printMu.Unlock()

	if index == 0 {
		fmt.Printf("\n\033[42;30m BEST MATCH \033[0m ")
	} else if index == 1 {
		fmt.Printf("\n\033[43;30m CLOSE ALTERNATIVE \033[0m ")
	} else {
		fmt.Printf("\n\033[45;37m RELEVANT RESULT \033[0m ")
	}

	fmt.Printf("\033[36m[%s]\033[0m \033[1m%s\033[0m (Score: %d)\n", res.Source, res.Title, res.Score)
	if res.URL != "" {
		fmt.Printf("\033[34m%s\033[0m\n", res.URL)
	}

	runes := []rune(res.Body)
	if len(runes) > 1000 {
		fmt.Println(string(runes[:1000]) + "...")
	} else {
		fmt.Println(res.Body)
	}

	if res.CodeSnippet != "" {
		fmt.Println("\033[42;30m Code snippet: \033[0m")
		codeLines := strings.Split(res.CodeSnippet, "\n")
		for i, line := range codeLines {
			if i > 20 {
				fmt.Println("\033[32m  ... [more in link]\033[0m")
				break
			}
			fmt.Printf("\033[32m  %s\033[0m\n", line)
		}
	}
	fmt.Println(strings.Repeat("-", 80))
}

func main() {
	initDatabases()

	reader := bufio.NewReader(os.Stdin)
	for {
		clear()
		fmt.Print("\033[33m")
		fmt.Println(`
██████╗  ██████╗ ██╗  ████████╗    ████████╗ ██████╗  ██████╗ ██╗
██╔══██╗██╔═══██╗██║  ╚══██╔══╝    ╚══██╔══╝██╔═══██╗██╔═══██╗██║
██████╔╝██║   ██║██║     ██║          ██║   ██║   ██║██║   ██║██║
██╔══██╗██║   ██║██║     ██║          ██║   ██║   ██║██║   ██║██║
██████╔╝╚██████╔╝███████╗██║          ██║   ╚██████╔╝╚██████╔╝███████╗
╚═════╝  ╚═════╝ ╚══════╝╚═╝          ╚═╝    ╚═════╝  ╚═════╝ ╚══════╝`)
		fmt.Print("\033[0m")
		fmt.Println("\n\033[32m[Version: BOLT Tool 2.1 Master]\033[0m")
		fmt.Println("Author: Coffee_Player | GitHub: https://github.com/CoffeePlayer")
		fmt.Println("\n[1] Auto Research (GitHub, StackOverflow, HackerNews, Dev.to, Wiki and more)")
		fmt.Println("[2] Target: Strict Code-Only (Discard results without code!)")
		fmt.Println("[3] Target: GitHub Repositories Only")
		fmt.Println("[4] Target: Stack Overflow Only")
		fmt.Println("[5] Target: Hacker News Only")
		fmt.Println("[6] Target: Wikipedia Only")
		fmt.Println("[7] \033[31m\033[0mYouTube Video Search")
		fmt.Println("[8] \033[35m\033[0mAI Research (Multi-Provider AI)")
		fmt.Println("[9] Exit")
		fmt.Println("")

		fmt.Print("\033[33mBolt > \033[0mSelect: ")

		selectionStr, _ := reader.ReadString('\n')
		selectionStr = strings.TrimSpace(selectionStr)

		if selectionStr == "9" {
			break
		}

		validOptions := map[string]bool{"1": true, "2": true, "3": true, "4": true, "5": true, "6": true, "7": true, "8": true}
		if !validOptions[selectionStr] {
			continue
		}

		fmt.Print("\033[33mBolt > \033[0mQuestion: ")
		rawQ, _ := reader.ReadString('\n')
		rawQ = strings.TrimSpace(rawQ)
		if rawQ == "" {
			continue
		}

		if len(rawQ) > 500 {
			fmt.Printf("\n\033[31m[ERROR]: Query exceeds 500 characters (Current: %d). Please shorten it.\033[0m\n", len(rawQ))
			fmt.Println("Press Enter to return...")
			_, _ = reader.ReadString('\n')
			continue
		}

		var sessionLang string
		rawQ, sessionLang = manualTranslateMenu(reader, rawQ)

		if selectionStr == "8" {
			correctedQ := autoCorrect(rawQ)
			englishQ := translateToEnglish(correctedQ)
			fetchAIResearch(englishQ)
			fmt.Println("\nPress Enter to return...")
			_, _ = reader.ReadString('\n')
			continue
		}

		strictCodeMode := (selectionStr == "2")
		isYouTubeOnly := (selectionStr == "7")

		correctedQ := autoCorrect(rawQ)
		complexity := classifyComplexity(correctedQ)

		var brain BrainResult
		if complexity != "EASY" {
			brain = preSearchBrain(correctedQ)
		} else {
			brain = BrainResult{
				EnglishQ:   fastTranslate(correctedQ),
				OptimizedQ: fastTranslate(correctedQ),
				Targets:    map[string]int{"WIKIPEDIA": 100},
				Insight:    "Fast translation active.",
			}
		}

		optimizedQ := brain.OptimizedQ

		fmt.Printf("\033[32mKeywords: %s\033[0m\n", optimizedQ)

		var targets []string
		type kv struct {
			K string
			V int
		}
		var sortedTargets []kv
		for k, v := range brain.Targets {
			sortedTargets = append(sortedTargets, kv{k, v})
		}
		sort.Slice(sortedTargets, func(i, j int) bool { return sortedTargets[i].V > sortedTargets[j].V })
		for _, item := range sortedTargets {
			if item.V > 25 {
				targets = append(targets, item.K)
			}
		}
		if len(targets) > 2 {
			targets = targets[:2]
		}

		if len(targets) == 0 {
			targets = []string{"STACK OVERFLOW", "GITHUB"}
		}

		multiQueries := generateMultiQueries(brain)
		ghQueries := generateGitHubQueries(optimizedQ)

		resultsChan := make(chan BoltResult, 500)
		var wg sync.WaitGroup

		isGitHubOnly := (selectionStr == "3")

		if isYouTubeOnly {
			wg.Add(1)
			go fetchYouTube(optimizedQ, resultsChan, &wg)
		} else if selectionStr == "1" || strictCodeMode {
		}

		executeTarget := func(target, query string) {
			priority := brain.Targets[target]
			wg.Add(1)
			switch target {
			case "STACK OVERFLOW":
				go fetchStackOverflow(query, resultsChan, &wg, priority)
			case "GITHUB":
				go fetchGitHub(query, resultsChan, &wg, priority)
			case "HACKER NEWS":
				go fetchHackerNews(query, resultsChan, &wg, priority)
			case "DEV.TO":
				go fetchDevTo(query, resultsChan, &wg, priority)
			case "WIKIPEDIA":
				go fetchWiki(query, resultsChan, &wg, priority)
			}
		}

		if isYouTubeOnly {
		} else if isGitHubOnly {
			for _, qVariant := range ghQueries {
				executeTarget("GITHUB", qVariant)
			}
		} else {
			ghFired := false
			for _, qVariant := range multiQueries {
				if selectionStr == "1" || strictCodeMode {
					for _, target := range targets {
						if target == "GITHUB" {
							if !ghFired {
								for _, ghV := range ghQueries {
									executeTarget("GITHUB", ghV)
								}
								ghFired = true
							}
						} else {
							executeTarget(target, qVariant)
						}
					}
				} else if selectionStr == "4" {
					executeTarget("STACK OVERFLOW", qVariant)
				} else if selectionStr == "5" {
					executeTarget("HACKER NEWS", qVariant)
				} else if selectionStr == "6" {
					executeTarget("WIKIPEDIA", qVariant)
				}
			}
		}

		var printWg sync.WaitGroup
		var finalResults []BoltResult
		printWg.Add(1)

		go func() {
			defer printWg.Done()
			seenURLs := make(map[string]bool)
			domainCounts := make(map[string]int)

			for res := range resultsChan {
				if res.Score < 0 {
					continue
				}
				if strictCodeMode && res.CodeSnippet == "" {
					continue
				}
				u, err := url.Parse(res.URL)
				domain := "unknown"
				if err == nil {
					domain = u.Host
				}

				if !seenURLs[res.URL] {
					if domainCounts[domain] >= 3 {
						res.Score -= 2000
					}
					finalResults = append(finalResults, res)
					seenURLs[res.URL] = true
					domainCounts[domain]++
				}
			}

			sort.Slice(finalResults, func(i, j int) bool {
				return finalResults[i].Score > finalResults[j].Score
			})

			limit := len(finalResults)
			maxResults := 2
			if limit > maxResults {
				limit = maxResults
			}
			for i := 0; i < limit; i++ {
				res := finalResults[i]
				printResultLive(res, i)

				if sessionLang != "" && sessionLang != "English" {
					translated := fastTranslateText(res.Body, sessionLang)
					fmt.Printf("\033[32m[%s]:\033[0m %s\n", strings.ToUpper(sessionLang), translated)
					fmt.Println(strings.Repeat("-", 80))
				}
			}

			if len(finalResults) == 0 {
				if strictCodeMode {
					fmt.Println("\033[31m[!] SEARCH FAILED. No code blocks found in search results.\033[0m")
				} else {
					fmt.Println("\033[31m[!] SEARCH FAILED. No relevant matches found after filtering.\033[0m")
				}
			}
		}()

		wg.Wait()
		close(resultsChan)
		printWg.Wait()

		fmt.Println("\nPress Enter to return...")
		_, _ = reader.ReadString('\n')
	}
}

func fastTranslateText(text string, targetLang string) string {
	if text == "" || targetLang == "" {
		return text
	}
	langCode := "en"
	switch strings.ToLower(targetLang) {
	case "polish", "polski":
		langCode = "pl"
	case "german", "niemiecki":
		langCode = "de"
	case "french", "francuski":
		langCode = "fr"
	case "spanish", "hiszpański":
		langCode = "es"
	}
	runes := []rune(text)
	chunkSize := 450
	var finalResult []string
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		translatedChunk := callMyMemory(chunk, langCode)
		finalResult = append(finalResult, translatedChunk)
	}
	return strings.Join(finalResult, " ")
}

func callMyMemory(cleanText, langCode string) string {
	u := "https://api.mymemory.translated.net/get?q=" + url.QueryEscape(cleanText) + "&langpair=en|" + langCode
	resp, err := httpClient.Get(u)
	if err != nil {
		return cleanText
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return cleanText
	}
	var res struct {
		ResponseData struct {
			TranslatedText string `json:"translatedText"`
		} `json:"responseData"`
		ResponseMessage string `json:"responseMessage"`
		ResponseStatus  int    `json:"responseStatus"`
	}
	if json.NewDecoder(resp.Body).Decode(&res) == nil && res.ResponseStatus == 200 && res.ResponseData.TranslatedText != "" {
		out := res.ResponseData.TranslatedText
		if strings.Contains(strings.ToUpper(out), "LIMIT EXCEEDED") || strings.Contains(strings.ToUpper(out), "MYMEMORY") {
			return cleanText + "..."
		}
		return out
	}
	return cleanText
}

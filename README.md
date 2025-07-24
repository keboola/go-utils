# Keboola Go Utils

- **deepcopy**
  - Deep copy and deep translate of a value, extended version of [hvoecking gist](https://gist.github.com/hvoecking/10772475).
  - Usage `import "github.com/keboola/go-utils/pkg/deepcopy"`.
- **diff**
  - Diff tool to copare two JSONs with wilcards
  - usage `import "github.com/keboola/go-utils/pkg/diff"`.
- **orderedmap**
  - Extended version of [iancoleman/orderedmap](https://github.com/iancoleman/orderedmap).
  - Usage `import "github.com/keboola/go-utils/pkg/orderedmap"`.
- **testproject**
  - Locking of Keboola Projects for E2E parallel tests.
  - Usage `import "github.com/keboola/go-utils/pkg/testproject"`.
- **wildcards**
  - Helper to compare text with wildcards in test.
  - Usage `import "github.com/keboola/go-utils/pkg/wildcards"`.
## Development

Clone the repository and run dev container:
```sh
docker-compose run --rm -u "$UID:$GID" --service-ports dev bash
```

Run lint and tests in container:
```sh
task lint
task tests
```

Run HTTP server with documentation:
```sh
task godoc
```

Open `http://localhost:6060/pkg/github.com/keboola/go-utils/pkg/` in browser.

## License

MIT licensed, see [LICENSE](./LICENSE) file.

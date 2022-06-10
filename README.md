# Go Utils
 
- **deepcopy**
  - Deep copy and deep translate of a value, extended version of [hvoecking gist](https://gist.github.com/hvoecking/10772475).
  - Usage `import "github.com/keboola/go-utils/pkg/deepcopy"`.
- **orderedmap**
  - Extended version of [iancoleman/orderedmap](https://github.com/iancoleman/orderedmap).
  - Usage `import "github.com/keboola/go-utils/pkg/orderedmap"`. 

## Development

Clone the repository and run dev container:
```sh
docker-compose run --rm -u "$UID:$GID" --service-ports dev bash
```

Run lint and tests in container:
```sh
make lint
make tests
```

Run HTTP server with documentation:
```sh
make godoc
```

Open `http://localhost:6060/pkg/github.com/keboola/go-utils/pkg/` in browser.

## License

MIT licensed, see [LICENSE](./LICENSE) file.

# xmigrate

Schema first DB migration tool for PostgreSQL.


## Getting Started

### Prerequisites
- Go 1.12+

### Installing
```
$ go get -u github.com/akito0107/xmigrate/cmd/xmigrate
```

### options
```$xslt
NAME:
   xmigrate - postgres db migration utility

USAGE:
   xmigrate [GLOBAL OPTIONS] [COMMANDS] [sub command options]

VERSION:
   0.0.0

COMMANDS:
     sync
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --host value                db host (default: "127.0.0.1")
   --port value, -p value      db host (default: "5432")
   --dbname value, -d value    dbname
   --password value, -W value  password
   --username value, -U value  db user name (default: "postgres")
   --verbose
   --help, -h                  show help
   --version, -v               print the version
```

## License
This project is licensed under the Apache License 2.0 License - see the [LICENSE](LICENSE) file for details

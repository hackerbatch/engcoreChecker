# EngCore Checker
A service for checking whether UBC EngCore is up. 

## Running

```bash
$ git clone https://github.com/hackerbatch/engcoreChecker.git
$ cd engcoreChecker
$ go get && go build
$ ./engcoreChecker # This will ask for a password to encrypt the encryption key.
2016/01/20 15:26:44 Generating new key ./db.key...
Key Password:
2015/01/20 15:26:53 Listening on :3000
```

## License
EngCore Checker is licensed under the Apache 2.0 license.

Made by [David Baldwynn](https://baldwynn.me).

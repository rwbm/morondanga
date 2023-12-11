# Morondanga

Do you usually find yourself copying and pasting pieces of code from other projects every time you start something new? Do you feel like always starting from scratch? Thinking about the logger, the database access library or the HTTP framwork? 

Well, Monrongan comes to solve this on some way. This is not a framework, but more like a set of libraries that provides a good starting point to build HTTP APIs. It has some default behaviour, and some extra modules that can be enabled or disabled by configuration.

It's a very opinionated one, and some decisions had to be made in order to it work, without spending too much time. The goal is to make your job easier, not harder. 

This is work in progress and more documentation will be added on the future, so please have some patience. 


## Usage

The simplest way yo use it is:

1. Create a configuration file named `config.yml`. You can use the provided in the root of the project as a stating point.

2. Create a new instance of the service:

```go
service, err := morondanga.NewService("")
if err != nil {
    panic(err)
}
```

The function `NewService()` accepts a string with the directory of the config.yml file. If this parameter is empty, the service will try to load if from the current directory and a config directory inside the current path.

Then, you can just run the service:

```go
if err = service.Run(); err != nil {
    if err != http.ErrServerClosed {
        panic(err)
    }
}
```

This will start a HTTP server listening on port 8080, asuming you didn't change the address in the config.yml file. It will also have a default health check handler that you can call:

```
$ curl http://localhost:8080/health
```

## Configruation yaml

The configuration file allows you to control the behaviour of the service. 


|Setting                |Default          |Notes                        |
|-----------------------|-----------------|-----------------------------|
|app.name               |`""`             | Application name |
|app.debug              |`false`          | Sets the service in debug mode |
|http.address           |`127.0.0.1:8080` | Address and port where the HTTP server listens |
|http.readTimeout       |`5 seconds`      | HTTP read timeout |
|http.writeTimeout      |`10 seconds`     | HTTP write timeout |
|http.idleTimeout       |`2 minutes`      | HTTP iddle timeout |
|http.customHealthCheck |`false`          | If sets to false, a default health check is used |
|http.jwtEnabled        |`false`          | Enable/disable a JWT configuration |
|http.jwtSigningKey     |`""`             | JWT signing key |
|database.enabled       |`false`          | Enables/disables the database integration |
|database.driver        |`""`             | Database driver. Currently supported: `mysql` |
|database.address       |`""`             | Database server address |
|database.user          |`""`             | Database username |
|database.password      |`""`             | Database password |
|database.database      |`""`             | Database name |

You can also define some custom entries. Those must be defined under the `custom` section. For example:

```yaml
custom:
  mycustomkey: "my custom value"
```

Custom keys can be accesed like this:


```go
myKeyValue, ok := service.Configuration().GetCustom("mycustomkey")
if !ok {
    fmt.Println("unknown configuration key")
}
```

Keep in mind that the configuration loader will convert any key value to lower case. For example, if you define an entry like `myCustomKey`, you'll have to use the key `mycustomkey`, or you won't find it. 
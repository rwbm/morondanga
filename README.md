# Morondanga

Do you usually find yourself copying and pasting pieces of code from other projects every time you start something new? Do you feel like always starting from scratch? Thinking about the logger, the database access library or the HTTP framwork? 

Well, Monrongan comes to solve this on some way. This is not a framework, but more like a set of libraries that provides a good starting point to build HTTP APIs. It has some default behaviour, and some extra modules that can be enabled or disabled by configuration.

It's a very opinionated one, and some decisions had to be made in order to it work, without spending too much time. The goal is to make your job easier, not harder. 

This is work in progress and more documentation will be added on the future, so please have some patience. 


## Usage

The simplest way yo use it is:

1. Get the library:

```
$ go get -u github.com/rwbm/morondanga
```

2. Create a configuration file named `config.yml`. You can use the provided in the root of the project as a stating point.

3. Create a new instance of the service:

```go
service, err := morondanga.NewService("")
if err != nil {
    panic(err)
}
```

The function `NewService()` accepts a string with the directory of the config.yml file. If this parameter is empty, the service will try to load if from the current directory and a config directory inside the current path.

4. Now can just run the service:

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

## Configuration yaml

The configuration file allows you to control the behaviour of the service. 


|Setting                  |Default          |Notes                        |
|-------------------------|-----------------|-----------------------------|
|`app.name`               |`"MyApp"` | Application name |
|`app.logLevel`           |`-1` | Logging levels: `-1=DEBUG`, `0=INFO`, `1=WARNING`, `2=ERROR` |
|`http.address`           |`127.0.0.1:8080` | Address and port where the HTTP server listens |
|`http.readTimeout`       |`5 seconds` | Maximum duration for reading the entire request, including the body |
|`http.writeTimeout`      |`10 seconds` | Maximum duration before timing out writes of the response |
|`http.idleTimeout`       |`2 minutes` | Maximum amount of time to wait for the next request when keep-alives are enabled |
|`http.customHealthCheck` |`false` | If sets to false, a default health check is used |
|`http.jwtEnabled`        |`false` | Enable/disable a JWT configuration |
|`http.jwtSigningKey`     |`"default-signing-key"` | JWT signing key. DON'T use the default value on production |
|`database.enabled`       |`false` | Enables/disables the database integration |
|`database.driver`        |`""` | Database driver. Currently supported: `mysql` |
|`database.address`       |`""` | Database server address |
|`database.user`          |`""` | Database username |
|`database.password`      |`""` | Database password |
|`database.database`      |`""` | Database name |

You can also define some custom entries. Those must be defined under the `custom` section. For example:

```yaml
custom:
  myCustomKey: "my custom value"
```

Custom keys can be accesed like this:


```go
myKeyValue, ok := service.Configuration().GetCustom("mycustomkey")
if !ok {
    fmt.Println("unknown configuration key")
}
```

Keep in mind that the configuration loader will convert any key value to lower case. For example, if you define an entry like `myCustomKey`, you'll have to use the key `mycustomkey` to retrieve it, or you won't be able to find it. 

What if want to add some extra sections to the config.yml file and I don't want to write my own logic to load the file? Well, we got you covered. Let's say that you want to have another section to the yaml, for example with the data to connect to Kafka topic:

```yaml
kafka:
    brokerAddress: "localhost:9092"
    topic: "your_topic_name"
```

You can define a custom Configuration structure composed of the config.Config and your additional structure:

```go
type MyCustomConfiguration struct {
	config.Config
	Kafka struct {
		BrokerAddress string
		Topic         string
	}
}
```

Then, you can use the function `NewServiceWithCustomConfiguration` to create an instance of the service with a custom configuration object:

```go
myCustomConfig := new(MyCustomConfiguration)
service, err := morondanga.NewServiceWithCustomConfiguration("", myCustomConfig)
if err != nil {
    panic(err)
}
```

You can now use `myCustomConfig` to access your custom fields (`myCustomConfig.Kafka.BrokerAddress`), as well as the standard service configuration. Keep in mind that if you make any change to this configuration object, it will also change the service configuration. This may or not be a good solution, but at least you don't have to write your own code to load those settings.

But, there's a subtle difference between the standard configuration file and a custom one. When using a custom configuration file, all the service related settings, have to be inside a section named `config`, that matches the name of field `config.Config`. Let's see an example:

```yaml
config:
  app:
    name: "MyApp"
    logLevel: -1

  http:
    address: 127.0.0.1:8080
    readTimeout: "5s"
    writeTimeout: "10s"
    idleTimeout: "2m"
    customHealthCheck: false
    jwtEnabled: true
    jwtSigningKey: "abcdefgh12345678"
    jwtTokenExpiration: "72h"

kafka:
    brokerAddress: "localhost:9092"
    topic: "your_topic_name"
```

## Components

To-do...

## Roadmap

I don't have a clear roadmap, but I would love to add some common components, like:

- Redis
- Kafka
- Other databases like Postgres or SQLite
- Access to AWS services


Feel free to contact me if there's something you want to add to this library, or better yet: send me a PR and I will be more than happy to review it 😎.
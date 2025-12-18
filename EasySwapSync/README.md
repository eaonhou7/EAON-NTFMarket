# EasySwapSync
EasySwapSync is a service that synchronizes EasySwap contract events from the blockchain to the database.

## Prerequisites
### Mysql & Redis
You should get your MYSQL & Redis running and create a database inside. MYSQL & Redis inside docker is recommended for local testing.
For example, if the machine is arm64 architecture, you can use the following docker-compose file to start mysql and redis.
```shell
docker-compose -f docker-compose-arm64.yml up -d
```

For more information about the table creation statement, see the SQL file in the db/migrations directory.

### Set Config file
Copy config/config.toml.example to config/config.toml. 
And modify the config file according to your environment, especially the mysql and redis connection information.
And set contract address in config file.

## Run
Run command below
```shell
go run main.go daemon
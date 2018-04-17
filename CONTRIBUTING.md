# Contributing guide


Development environment
------------
* **Step 1.** Get Golang
```bash
brew install go
brew install glide

export GOPATH=~/workspace/go
git clone git@github.com:MysteriumNetwork/node.git $GOPATH/src/github.com/mysterium/node
cd $GOPATH/src/github.com/mysterium/node
```

* **Step 2.** Compile code
```bash
glide install
go build github.com/mysterium/node
```

* **Step 3.** Prepare configuration

Enter `MYSTERIUM_API_URL` and value of running [api](https://github.com/MysteriumNetwork/api) instance

```bash
cp .env_example .env
vim .env
```

For example if your [api](https://github.com/MysteriumNetwork/api) is listening on `your.hostname.com:8001`, then the content of the `.env` file should look like this

```
MYSTERIUM_API_URL=http://your.hostname.com:8001/v1
NATS_SERVER_IP=your.hostname.com
```

Running
------------
``` bash
# Start communication broker
docker-compose up broker

# Start node
bin/server_build
bin/server_run

# Client connects to node
bin/client_build
bin/client_run
```

Dependency management
------------
* Install project's frozen packages
```bash
glide install
glide install --force
```

* Add new package to project
```bash
glide get github.com/ccding/go-stun
```

* Update package in project
```bash
vim glide.yaml
glide update
```


Debian packaging
------------
* **Step 1.** Get FPM tool
See http://fpm.readthedocs.io/en/latest/installing.html

```bash
brew install gnu-tar
gem install --no-ri --no-rdoc fpm
```

* **Step 2.** Get Debber tool
See https://github.com/debber/debber-v0.3

```bash
go get github.com/debber/debber-v0.3/cmd/...
```

* **Step 3.** Build .deb package
```bash
bin/server_package_debian 0.0.6 amd64
bin/client_package_debian 0.0.6 amd64
```

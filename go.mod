module github.com/mysteriumnetwork/node

go 1.13

replace golang.zx2c4.com/wireguard => github.com/mysteriumnetwork/wireguard-go v0.0.0-20191114114228-1a3c4386eb23

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/DataDog/zstd v1.4.1 // indirect
	github.com/Sereal/Sereal v0.0.0-20190618215532-0b8ac451a863 // indirect
	github.com/allegro/bigcache v1.2.0 // indirect
	github.com/aristanetworks/goarista v0.0.0-20180809135256-70ad3c3262ad // indirect
	github.com/arthurkiller/rollingwriter v1.1.1
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05
	github.com/asdine/storm v2.1.2+incompatible
	github.com/aws/aws-sdk-go-v2 v0.15.0
	github.com/btcsuite/btcd v0.0.0-20180810000619-f899737d7f27 // indirect
	github.com/cespare/cp v1.1.1 // indirect
	github.com/cheggaaa/pb/v3 v3.0.1
	github.com/chzyer/logex v1.1.10 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/chzyer/test v0.0.0-20180213035817-a1ea475d72b1 // indirect
	github.com/deckarep/golang-set v0.0.0-20180603214616-504e848d77ea // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/ethereum/go-ethereum v1.8.23
	github.com/fjl/memsize v0.0.0-20190710130421-bcb5799ab5e5 // indirect
	github.com/frankban/quicktest v1.5.0 // indirect
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-gonic/gin v1.4.0
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/protobuf v1.3.2
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.5.4
	github.com/huin/goupnp v1.0.0
	github.com/jackpal/gateway v1.0.5
	github.com/jackpal/go-nat-pmp v1.0.1 // indirect
	github.com/julienschmidt/httprouter v1.2.0
	github.com/karalabe/hid v1.0.0 // indirect
	github.com/koron/go-ssdp v0.0.0-20180514024734-4a0ed625a78b
	github.com/magefile/mage v1.9.0
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/miekg/dns v1.0.12
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mysteriumnetwork/feedback v1.1.1
	github.com/mysteriumnetwork/go-ci v0.0.0-20190917134659-78a29f230467
	github.com/mysteriumnetwork/go-dvpn-web v0.0.0-20191125133122-c2bab4ca5537
	github.com/mysteriumnetwork/go-openvpn v0.0.18
	github.com/mysteriumnetwork/go-wondershaper v1.0.0
	github.com/mysteriumnetwork/metrics v0.0.0-20191002053948-084a00d6c6b2
	github.com/mysteriumnetwork/payments v0.0.11-0.20190809092009-003973d4b083
	github.com/nats-io/gnatsd v1.4.1 // indirect
	github.com/nats-io/go-nats v1.4.0
	github.com/nats-io/nuid v1.0.1-0.20180712044959-3024a71c3cbe // indirect
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/oleksandr/bonjour v0.0.0-20160508152359-5dcf00d8b228
	github.com/onsi/ginkgo v1.10.2 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/oschwald/geoip2-golang v1.1.0
	github.com/oschwald/maxminddb-golang v1.2.1 // indirect
	github.com/pierrec/lz4 v2.3.0+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/prometheus v1.7.2-0.20170814170113-3101606756c5 // indirect
	github.com/rjeczalik/notify v0.9.2 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/rs/cors v1.5.1-0.20180731071213-15587285ef6b // indirect
	github.com/rs/zerolog v1.16.0
	github.com/songgao/water v0.0.0-20190112225332-f6122f5b2fbd
	github.com/spf13/cast v1.3.0
	github.com/stretchr/testify v1.4.0
	github.com/syndtr/goleveldb v0.0.0-20180708030551-c4c61651e9e3 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	golang.org/x/crypto v0.0.0-20191122220453-ac88ee75c92c
	golang.org/x/net v0.0.0-20191125084936-ffdde1057850
	golang.org/x/sys v0.0.0-20191120155948-bd437916bb0e
	golang.zx2c4.com/wireguard v0.0.20191012
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20190515223858-5ec88494b814
	google.golang.org/appengine v1.6.5 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1 // indirect
	gopkg.in/urfave/cli.v1 v1.20.0
	gopkg.in/yaml.v2 v2.2.4 // indirect
)

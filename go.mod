module github.com/flomesh-io/fsm

go 1.24

require (
	github.com/AlekSi/gocov-xml v1.1.0
	github.com/Azure/go-autorest/autorest/to v0.4.1
	github.com/axw/gocov v1.2.1
	github.com/blang/semver v3.5.1+incompatible
	github.com/briandowns/spinner v1.23.2
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cert-manager/cert-manager v1.17.1
	github.com/cilium/ebpf v0.17.3
	github.com/cskr/pubsub v1.0.2
	github.com/deckarep/golang-set v1.8.0
	github.com/deckarep/golang-set/v2 v2.8.0
	github.com/docker/distribution v2.8.3+incompatible
	github.com/docker/docker v27.5.0+incompatible
	github.com/docker/go-connections v0.5.0
	github.com/dubbogo/go-zookeeper v1.0.3
	github.com/dustin/go-humanize v1.0.1
	github.com/fatih/color v1.18.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/gin-gonic/gin v1.10.0
	github.com/go-co-op/gocron v1.37.0
	github.com/go-logr/logr v1.4.2
	github.com/go-logr/zerologr v1.2.3
	github.com/go-resty/resty/v2 v2.16.5
	github.com/go-sql-driver/mysql v1.9.0
	github.com/gobwas/glob v0.2.3
	github.com/golang/mock v1.6.0
	github.com/golangci/golangci-lint v1.64.7
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/hashicorp/consul/api v1.31.2
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-version v1.7.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/hashicorp/vault v1.19.0
	github.com/hashicorp/vault/api v1.16.0
	github.com/hashicorp/vault/sdk v0.15.2
	github.com/hudl/fargo v1.4.0
	github.com/jinzhu/copier v0.4.0
	github.com/jonboulle/clockwork v0.5.0
	github.com/jstemmer/go-junit-report v1.0.0
	github.com/k3d-io/k3d/v5 v5.8.3
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/libp2p/go-netroute v0.2.2
	github.com/matm/gocov-html v1.4.0
	github.com/mdlayher/arp v0.0.0-20220512170110-6706a2966875
	github.com/mdlayher/ethernet v0.0.0-20220221185849-529eae5b6118
	github.com/mholt/archiver/v3 v3.5.1
	github.com/miekg/dns v1.1.63
	github.com/mitchellh/gox v1.0.1
	github.com/mitchellh/hashstructure/v2 v2.0.2
	github.com/nacos-group/nacos-sdk-go/v2 v2.2.9
	github.com/norwoodj/helm-docs v1.4.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.36.2
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/pkg/errors v0.9.1
	github.com/prometheus-community/pro-bing v0.6.1
	github.com/prometheus/client_golang v1.21.1
	github.com/prometheus/common v0.63.0
	github.com/rs/zerolog v1.33.0
	github.com/servicemeshinterface/smi-sdk-go v0.5.0
	github.com/sethvargo/go-retry v0.3.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.20.0
	github.com/stretchr/testify v1.10.0
	github.com/tidwall/gjson v1.18.0
	github.com/tidwall/sjson v1.2.5
	github.com/vishvananda/netlink v1.3.1-0.20250206174618-62fb240731fa
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394
	golang.org/x/net v0.37.0
	golang.org/x/sync v0.12.0
	golang.org/x/sys v0.31.0
	golang.org/x/time v0.11.0
	gomodules.xyz/jsonpatch/v2 v2.5.0
	google.golang.org/grpc v1.71.0
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/mysql v1.5.7
	gorm.io/gorm v1.25.12
	helm.sh/helm/v3 v3.17.2
	k8s.io/api v0.32.3
	k8s.io/apiextensions-apiserver v0.32.3
	k8s.io/apimachinery v0.32.3
	k8s.io/cli-runtime v0.32.3
	k8s.io/client-go v0.32.3
	k8s.io/code-generator v0.32.3
	k8s.io/gengo/v2 v2.0.0-20250207200755-1244d31929d7
	k8s.io/klog/v2 v2.130.1
	k8s.io/kubectl v0.32.3
	k8s.io/kubernetes v1.32.3
	k8s.io/utils v0.0.0-20241210054802-24370beab758
	sigs.k8s.io/controller-runtime v0.20.3
	sigs.k8s.io/gateway-api v1.2.1
	sigs.k8s.io/gwctl v0.1.0
	sigs.k8s.io/kind v0.27.0
	sigs.k8s.io/yaml v1.4.0
)

require (
	4d63.com/gocheckcompilerdirectives v1.3.0 // indirect
	4d63.com/gochecknoglobals v0.2.2 // indirect
	al.essio.dev/pkg/shellescape v1.5.1 // indirect
	cloud.google.com/go v0.116.0 // indirect
	cloud.google.com/go/auth v0.15.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.7 // indirect
	cloud.google.com/go/cloudsqlconn v1.4.3 // indirect
	cloud.google.com/go/compute/metadata v0.6.0 // indirect
	cloud.google.com/go/iam v1.2.2 // indirect
	cloud.google.com/go/kms v1.20.1 // indirect
	cloud.google.com/go/longrunning v0.6.2 // indirect
	cloud.google.com/go/monitoring v1.21.2 // indirect
	dario.cat/mergo v1.0.1 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/4meepo/tagalign v1.4.2 // indirect
	github.com/Abirdcfly/dupword v0.1.3 // indirect
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20230811130428-ced1acdcaa24 // indirect
	github.com/Antonboom/errname v1.0.0 // indirect
	github.com/Antonboom/nilnil v1.0.1 // indirect
	github.com/Antonboom/testifylint v1.5.2 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.17.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.8.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys v0.10.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/internal v0.7.1 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.29 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.23 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.12 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.6 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.3.3 // indirect
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	github.com/Crocmagnon/fatcontext v0.7.1 // indirect
	github.com/DataDog/datadog-go v3.2.0+incompatible // indirect
	github.com/Djarvur/go-err113 v0.0.0-20210108212216-aea10b59be24 // indirect
	github.com/GaijinEntertainment/go-exhaustruct/v3 v3.3.1 // indirect
	github.com/Jeffail/gabs/v2 v2.1.0 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/semver/v3 v3.3.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/Masterminds/squirrel v1.5.4 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/OpenPeeDeeP/depguard/v2 v2.2.1 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20230828082145-3c4c8a2d2371 // indirect
	github.com/alecthomas/go-check-sumtype v0.3.1 // indirect
	github.com/alexkohler/nakedret/v2 v2.0.5 // indirect
	github.com/alexkohler/prealloc v1.0.0 // indirect
	github.com/alibabacloud-go/alibabacloud-gateway-pop v0.0.6 // indirect
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/darabonba-array v0.1.0 // indirect
	github.com/alibabacloud-go/darabonba-encode-util v0.0.2 // indirect
	github.com/alibabacloud-go/darabonba-map v0.0.2 // indirect
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.0.10 // indirect
	github.com/alibabacloud-go/darabonba-signature-util v0.0.7 // indirect
	github.com/alibabacloud-go/darabonba-string v1.0.2 // indirect
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/alibabacloud-go/endpoint-util v1.1.0 // indirect
	github.com/alibabacloud-go/kms-20160120/v3 v3.2.3 // indirect
	github.com/alibabacloud-go/openapi-util v0.1.0 // indirect
	github.com/alibabacloud-go/tea v1.2.2 // indirect
	github.com/alibabacloud-go/tea-utils v1.4.4 // indirect
	github.com/alibabacloud-go/tea-utils/v2 v2.0.7 // indirect
	github.com/alibabacloud-go/tea-xml v1.1.3 // indirect
	github.com/alingse/asasalint v0.0.11 // indirect
	github.com/alingse/nilnesserr v0.1.2 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.63.84 // indirect
	github.com/aliyun/alibabacloud-dkms-gcs-go-sdk v0.5.1 // indirect
	github.com/aliyun/alibabacloud-dkms-transfer-go-sdk v0.1.8 // indirect
	github.com/aliyun/aliyun-secretsmanager-client-go v1.1.5 // indirect
	github.com/aliyun/credentials-go v1.4.3 // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/ashanbrown/forbidigo v1.6.0 // indirect
	github.com/ashanbrown/makezero v1.2.0 // indirect
	github.com/aws/aws-sdk-go v1.55.6 // indirect
	github.com/axiomhq/hyperloglog v0.0.0-20220105174342-98591331716a // indirect
	github.com/benbjohnson/immutable v0.4.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bgentry/speakeasy v0.2.0 // indirect
	github.com/bkielbasa/cyclop v1.2.3 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/blizzy78/varnamelen v0.8.0 // indirect
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/bombsimon/wsl/v4 v4.5.0 // indirect
	github.com/boombuler/barcode v1.0.1 // indirect
	github.com/breml/bidichk v0.3.2 // indirect
	github.com/breml/errchkjson v0.4.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/butuzov/ireturn v0.3.1 // indirect
	github.com/butuzov/mirror v1.3.0 // indirect
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/catenacyber/perfsprint v0.8.2 // indirect
	github.com/ccojocar/zxcvbn-go v1.0.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/charithe/durationcheck v0.0.10 // indirect
	github.com/chavacava/garif v0.1.0 // indirect
	github.com/circonus-labs/circonus-gometrics v2.3.1+incompatible // indirect
	github.com/circonus-labs/circonusllhist v0.1.3 // indirect
	github.com/ckaznocha/intrange v0.3.0 // indirect
	github.com/clbanning/mxj v1.8.4 // indirect
	github.com/clbanning/mxj/v2 v2.5.5 // indirect
	github.com/cloudflare/circl v1.5.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/containerd/containerd v1.7.24 // indirect
	github.com/containerd/errdefs v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.14.3 // indirect
	github.com/coreos/etcd v3.3.27+incompatible // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/pkg v0.0.0-20220810130054-c7d1c02cb6cf // indirect
	github.com/curioswitch/go-reassign v0.3.0 // indirect
	github.com/cyphar/filepath-securejoin v0.3.6 // indirect
	github.com/daixiang0/gci v0.13.5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/denis-tingaikin/go-header v0.5.0 // indirect
	github.com/denverdino/aliyungo v0.0.0-20190125010748-a747050bb1ba // indirect
	github.com/dgryski/go-metro v0.0.0-20180109044635-280f6062b5bc // indirect
	github.com/digitalocean/godo v1.125.0 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v27.2.1+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.1 // indirect
	github.com/docker/go v1.5.1-1.0.20160303222718-d30aec9fd63c // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dsnet/compress v0.0.2-0.20210315054119-f66993602bf5 // indirect
	github.com/duosecurity/duo_api_golang v0.0.0-20190308151101-6c680f768e74 // indirect
	github.com/emicklei/go-restful/v3 v3.12.1 // indirect
	github.com/ettle/strcase v0.2.0 // indirect
	github.com/evanphx/json-patch v5.9.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/firefart/nonamedreturns v1.0.5 // indirect
	github.com/flopp/go-findfont v0.1.0 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/franela/goreq v0.0.0-20171204163338-bcd34c9993f8 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/fvbommel/sortorder v1.1.0 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/fzipp/gocyclo v0.6.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.7 // indirect
	github.com/gammazero/deque v0.2.1 // indirect
	github.com/gammazero/workerpool v1.1.3 // indirect
	github.com/ghostiam/protogetter v0.3.9 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-critic/go-critic v0.12.0 // indirect
	github.com/go-errors/errors v1.5.1 // indirect
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/go-ozzo/ozzo-validation v3.6.0+incompatible // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/go-test/deep v1.1.1 // indirect
	github.com/go-toolsmith/astcast v1.1.0 // indirect
	github.com/go-toolsmith/astcopy v1.1.0 // indirect
	github.com/go-toolsmith/astequal v1.2.0 // indirect
	github.com/go-toolsmith/astfmt v1.1.0 // indirect
	github.com/go-toolsmith/astp v1.1.0 // indirect
	github.com/go-toolsmith/strparse v1.1.0 // indirect
	github.com/go-toolsmith/typep v1.1.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/go-xmlfmt/xmlfmt v1.1.3 // indirect
	github.com/goccy/go-graphviz v0.2.9 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/golangci/dupl v0.0.0-20250308024227-f665c8d69b32 // indirect
	github.com/golangci/go-printf-func-name v0.1.0 // indirect
	github.com/golangci/gofmt v0.0.0-20250106114630-d62b90e6713d // indirect
	github.com/golangci/misspell v0.6.0 // indirect
	github.com/golangci/plugin-module-register v0.1.1 // indirect
	github.com/golangci/revgrep v0.8.0 // indirect
	github.com/golangci/unconvert v0.0.0-20240309020433-c5143eacb3ed // indirect
	github.com/goodhosts/hostsfile v0.1.6 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/certificate-transparency-go v1.3.1 // indirect
	github.com/google/gnostic-models v0.6.9 // indirect
	github.com/google/go-containerregistry v0.19.1 // indirect
	github.com/google/go-metrics-stackdriver v0.2.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/gopacket v1.1.19 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/safetext v0.0.0-20220905092116-b49f7bc46da2 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/googleapis/gax-go/v2 v2.14.1 // indirect
	github.com/gophercloud/gophercloud v0.3.0 // indirect
	github.com/gordonklaus/ineffassign v0.1.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/gostaticanalysis/analysisutil v0.7.1 // indirect
	github.com/gostaticanalysis/comment v1.5.0 // indirect
	github.com/gostaticanalysis/forcetypeassert v0.2.0 // indirect
	github.com/gostaticanalysis/nilerr v0.1.1 // indirect
	github.com/gosuri/uitable v0.0.4 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.25.1 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/hashicorp/cli v1.1.6 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/eventlogger v0.2.10 // indirect
	github.com/hashicorp/go-bexpr v0.1.12 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-discover v0.0.0-20220714221025-1c234a67149a // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-hmac-drbg v0.0.0-20210916214228-a6e5a68489f6 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-immutable-radix/v2 v2.1.0 // indirect
	github.com/hashicorp/go-kms-wrapping/entropy/v2 v2.0.1 // indirect
	github.com/hashicorp/go-kms-wrapping/v2 v2.0.18 // indirect
	github.com/hashicorp/go-kms-wrapping/wrappers/aead/v2 v2.0.10 // indirect
	github.com/hashicorp/go-kms-wrapping/wrappers/alicloudkms/v2 v2.0.4 // indirect
	github.com/hashicorp/go-kms-wrapping/wrappers/awskms/v2 v2.0.11 // indirect
	github.com/hashicorp/go-kms-wrapping/wrappers/azurekeyvault/v2 v2.0.14 // indirect
	github.com/hashicorp/go-kms-wrapping/wrappers/gcpckms/v2 v2.0.13 // indirect
	github.com/hashicorp/go-kms-wrapping/wrappers/ocikms/v2 v2.0.9 // indirect
	github.com/hashicorp/go-kms-wrapping/wrappers/transit/v2 v2.0.13 // indirect
	github.com/hashicorp/go-memdb v1.3.4 // indirect
	github.com/hashicorp/go-msgpack/v2 v2.1.2 // indirect
	github.com/hashicorp/go-plugin v1.6.1 // indirect
	github.com/hashicorp/go-raftchunking v0.7.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/awsutil v0.3.0 // indirect
	github.com/hashicorp/go-secure-stdlib/base62 v0.1.2 // indirect
	github.com/hashicorp/go-secure-stdlib/cryptoutil v0.1.1 // indirect
	github.com/hashicorp/go-secure-stdlib/mlock v0.1.3 // indirect
	github.com/hashicorp/go-secure-stdlib/nonceutil v0.1.0 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.9 // indirect
	github.com/hashicorp/go-secure-stdlib/permitpool v1.0.0 // indirect
	github.com/hashicorp/go-secure-stdlib/plugincontainer v0.4.1 // indirect
	github.com/hashicorp/go-secure-stdlib/reloadutil v0.1.1 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-secure-stdlib/tlsutil v0.1.3 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/go-syslog v1.0.0 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/hashicorp/hcp-sdk-go v0.101.0 // indirect
	github.com/hashicorp/mdns v1.0.4 // indirect
	github.com/hashicorp/raft v1.7.1 // indirect
	github.com/hashicorp/raft-autopilot v0.3.0 // indirect
	github.com/hashicorp/raft-boltdb/v2 v2.3.0 // indirect
	github.com/hashicorp/raft-snapshot v1.0.4 // indirect
	github.com/hashicorp/raft-wal v0.4.0 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/hashicorp/vault-plugin-secrets-kv v0.21.0 // indirect
	github.com/hashicorp/vic v1.5.1-0.20190403131502-bbfe86ec9443 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/hexops/gotextdiff v1.0.3 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.14.3 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgtype v1.14.3 // indirect
	github.com/jackc/pgx/v4 v4.18.3 // indirect
	github.com/jefferai/isbadcipher v0.0.0-20190226160619-51d2077c035f // indirect
	github.com/jefferai/jsonx v1.0.1 // indirect
	github.com/jgautheron/goconst v1.7.1 // indirect
	github.com/jingyugao/rowserrcheck v1.1.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jjti/go-spancheck v0.6.4 // indirect
	github.com/jmespath/go-jmespath v0.4.1-0.20220621161143-b0104c826a24 // indirect
	github.com/jmoiron/sqlx v1.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/josharian/native v1.1.0 // indirect
	github.com/joshlf/go-acl v0.0.0-20200411065538-eae00ae38531 // indirect
	github.com/joyent/triton-go v1.7.1-0.20200416154420-6801d15b779f // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/julz/importas v0.2.0 // indirect
	github.com/karamaru-alpha/copyloopvar v1.2.1 // indirect
	github.com/kisielk/errcheck v1.9.0 // indirect
	github.com/kkHAIKE/contextcheck v1.1.6 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/klauspost/pgzip v1.2.5 // indirect
	github.com/kulti/thelper v0.6.3 // indirect
	github.com/kunwardeep/paralleltest v1.0.10 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/lasiar/canonicalheader v1.1.2 // indirect
	github.com/ldez/exptostd v0.4.2 // indirect
	github.com/ldez/gomoddirectives v0.6.1 // indirect
	github.com/ldez/grignotin v0.9.0 // indirect
	github.com/ldez/tagliatelle v0.7.1 // indirect
	github.com/ldez/usetesting v0.4.2 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/leonklingele/grouper v1.1.2 // indirect
	github.com/lestrrat-go/backoff/v2 v2.0.8 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/jwx v1.2.29 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/linode/linodego v0.10.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/macabu/inamedparam v0.1.3 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/maratori/testableexamples v1.0.0 // indirect
	github.com/maratori/testpackage v1.1.1 // indirect
	github.com/matoous/godox v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mdlayher/packet v1.1.2 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/mgechev/revive v1.7.0 // indirect
	github.com/miekg/pkcs11 v1.1.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/iochan v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/pointerstructure v1.2.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/spdystream v0.5.0 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/user v0.3.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/moricho/tparallel v0.3.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/nakabonne/nestif v0.3.1 // indirect
	github.com/nicolai86/scaleway-sdk v1.10.2-0.20180628010248-798f60e20bb2 // indirect
	github.com/nishanths/exhaustive v0.12.0 // indirect
	github.com/nishanths/predeclared v0.2.2 // indirect
	github.com/nunnatsa/ginkgolinter v0.19.1 // indirect
	github.com/nwaples/rardecode v1.1.2 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/okta/okta-sdk-golang/v5 v5.0.2 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/opentracing/opentracing-go v1.2.1-0.20220228012449-10b1cf09e00b // indirect
	github.com/oracle/oci-go-sdk/v60 v60.0.0 // indirect
	github.com/orcaman/concurrent-map v0.0.0-20210501183033-44dafcb38ecc // indirect
	github.com/packethost/packngo v0.1.1-0.20180711074735-b9cb5096f54c // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/petermattis/goid v0.0.0-20240813172612-4fcff4a6cae7 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pires/go-proxyproto v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/polyfloyd/go-errorlint v1.7.1 // indirect
	github.com/posener/complete v1.2.3 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/pquerna/otp v1.2.1-0.20191009055518-468c2dd2b58d // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/quasilyte/go-ruleguard v0.4.3-0.20240823090925-0fe6f58b47b1 // indirect
	github.com/quasilyte/go-ruleguard/dsl v0.3.22 // indirect
	github.com/quasilyte/gogrep v0.5.0 // indirect
	github.com/quasilyte/regex/syntax v0.0.0-20210819130434-b3f0c404a727 // indirect
	github.com/quasilyte/stdinfo v0.0.0-20220114132959-f7386bf02567 // indirect
	github.com/raeperd/recvcheck v0.2.0 // indirect
	github.com/rancher/wharfie v0.6.2 // indirect
	github.com/rboyer/safeio v0.2.1 // indirect
	github.com/renier/xmlrpc v0.0.0-20170708154548-ce4a1a486c03 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/rotisserie/eris v0.5.4 // indirect
	github.com/rubenv/sql-migrate v1.7.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryancurrah/gomodguard v1.3.5 // indirect
	github.com/ryanrolds/sqlclosecheck v0.5.1 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sanposhiho/wastedassign/v2 v2.1.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.1 // indirect
	github.com/sasha-s/go-deadlock v0.3.5 // indirect
	github.com/sashamelentyev/interfacebloat v1.1.0 // indirect
	github.com/sashamelentyev/usestdlibvars v1.28.0 // indirect
	github.com/securego/gosec/v2 v2.22.2 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/sethvargo/go-limiter v0.7.1 // indirect
	github.com/shirou/gopsutil/v3 v3.24.5 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sivchari/containedctx v1.0.3 // indirect
	github.com/sivchari/tenv v1.12.1 // indirect
	github.com/softlayer/softlayer-go v0.0.0-20180806151055-260589d94c7d // indirect
	github.com/sonatard/noctx v0.1.0 // indirect
	github.com/sony/gobreaker v0.5.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/sourcegraph/go-diff v0.7.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/ssgreg/nlreturn/v2 v2.2.1 // indirect
	github.com/stbenjam/no-sprintf-host-port v0.2.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tdakkota/asciicheck v0.4.1 // indirect
	github.com/tencentcloud/tencentcloud-sdk-go v1.0.162 // indirect
	github.com/tetafro/godot v1.5.0 // indirect
	github.com/tetratelabs/wazero v1.8.1 // indirect
	github.com/theupdateframework/notary v0.7.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/timakin/bodyclose v0.0.0-20241017074812-ed6a65f985e3 // indirect
	github.com/timonwong/loggercheck v0.10.1 // indirect
	github.com/tink-crypto/tink-go/v2 v2.2.0 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/tomarrell/wrapcheck/v2 v2.10.0 // indirect
	github.com/tommy-muehle/go-mnd/v2 v2.5.1 // indirect
	github.com/tv42/httpunix v0.0.0-20191220191345-2ba4b9c3382c // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/ultraware/funlen v0.2.0 // indirect
	github.com/ultraware/whitespace v0.2.0 // indirect
	github.com/uudashr/gocognit v1.2.0 // indirect
	github.com/uudashr/iface v1.3.1 // indirect
	github.com/vbatts/tar-split v0.11.3 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	github.com/vmware/govmomi v0.30.6 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xen0n/gosmopolitan v1.2.2 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	github.com/yagipy/maintidx v1.0.0 // indirect
	github.com/yeya24/promlinter v0.3.0 // indirect
	github.com/ykadowak/zerologlint v0.1.5 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	gitlab.com/bosi/decorder v0.4.2 // indirect
	go-simpler.org/musttag v0.13.0 // indirect
	go-simpler.org/sloglint v0.9.0 // indirect
	go.etcd.io/bbolt v1.4.0-beta.0 // indirect
	go.mongodb.org/mongo-driver v1.17.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.59.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.59.0 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.33.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.33.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	go.opentelemetry.io/otel/sdk v1.34.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.34.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	go.opentelemetry.io/proto/otlp v1.4.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go4.org/netipx v0.0.0-20231129151722-fdeea329fbba // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/exp/typeparams v0.0.0-20250210185358-939b2ce775ac // indirect
	golang.org/x/image v0.21.0 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/oauth2 v0.27.0 // indirect
	golang.org/x/term v0.30.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	google.golang.org/api v0.223.0 // indirect
	google.golang.org/genproto v0.0.0-20241118233622-e639e219e697 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250106144421-5f5ef82da422 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250219182151-9fdb1cabc7b2 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/gcfg.v1 v1.2.3 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/resty.v1 v1.12.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	honnef.co/go/tools v0.6.1 // indirect
	k8s.io/apiserver v0.32.3 // indirect
	k8s.io/component-base v0.32.3 // indirect
	k8s.io/component-helpers v0.32.3 // indirect
	k8s.io/controller-manager v0.0.0 // indirect
	k8s.io/helm v2.14.3+incompatible // indirect
	k8s.io/kube-openapi v0.0.0-20241212222426-2c72e554b1e7 // indirect
	mvdan.cc/gofumpt v0.7.0 // indirect
	mvdan.cc/unparam v0.0.0-20240528143540-8a5130ca722f // indirect
	oras.land/oras-go v1.2.5 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/kustomize/api v0.18.0 // indirect
	sigs.k8s.io/kustomize/kyaml v0.18.1 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.5.0 // indirect
)

replace (
	github.com/distribution/reference => github.com/distribution/reference v0.5.0
	github.com/pires/go-proxyproto v1.0.0 => github.com/peteski22/go-proxyproto v1.0.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.32.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.32.3
	k8s.io/controller-manager => k8s.io/controller-manager v0.32.3
	k8s.io/cri-api => k8s.io/cri-api v0.32.3
	k8s.io/cri-client => k8s.io/cri-client v0.32.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.32.3
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.32.3
	k8s.io/endpointslice => k8s.io/endpointslice v0.32.3
	k8s.io/externaljwt => k8s.io/externaljwt v0.32.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.32.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.32.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.32.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.32.3
	k8s.io/kubelet => k8s.io/kubelet v0.32.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.32.3
	k8s.io/metrics => k8s.io/metrics v0.32.3
	k8s.io/mount-utils => k8s.io/mount-utils v0.32.3
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.32.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.32.3
	k8s.io/utils/net => k8s.io/utils/net v0.32.3
)

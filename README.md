# 从 0 - 1 实现 CRD (& Controller)

集群中存在两种资源 Seller 和 Buyer

- Seller 通过售卖一定价格（price）和数量（amount）的产品（name）进行获利（money）
- Buyer 会根据最高买入价（price）和需求量（amount）来选择一个或多个渠道来买入产品（name）

e.g. 卖苹果的以 100 元每个的价格售卖 10 个苹果，如果有买家出价超过 100，则以买入价来结算总价值

## Handcraft (with client-go)

### Step 1 - 创建 CRD 资源

- 声明 CRD 资源

相当于注册一个 API 接口，让 K8S API Server 识别并处理请求

```yaml
# artifacts/crd-seller.yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: sellers.playgroundk8scrd.anddd7.github.com
spec:
  group: playgroundk8scrd.anddd7.github.com
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                name:
                  type: string
                amount:
                  type: integer
                price:
                  type: number
                money:
                  type: number
  scope: Namespaced
  names:
    plural: sellers
    singular: seller
    kind: Seller
    shortNames:
      - seller
# artifacts/crd-buyer.yaml
...
              properties:
                name:
                  type: string
                amount:
                  type: integer
                price:
                  type: number
  scope: Namespaced
  names:
    plural: buyers
    singular: buyer
    kind: Buyer
    shortNames:
      - buyer
```

- 声明 Seller 和 Buyer 资源

通过 Yaml 创建“资源”，就像创建 Pod、Deployment 一样

```yaml
# artifacts/sellers.yaml
apiVersion: "playgroundk8scrd.anddd7.github.com/v1alpha1"
kind: Seller
metadata:
  name: seller-apple
spec:
  name: apple
  amount: 10
  price: 100
  money: 0
# artifacts/buyers.yaml
apiVersion: "playgroundk8scrd.anddd7.github.com/v1alpha1"
kind: Buyer
metadata:
  name: buyer-apple
spec:
  name: apple
  amount: 10
  price: 100
```

- 测试和验证

Apply 所有的资源到集群里

```sh
# apply crd
k apply -f artifacts/crd-seller.yaml -f artifacts/crd-buyer.yaml

k get crd sellers.playgroundk8scrd.anddd7.github.com
k get crd buyers.playgroundk8scrd.anddd7.github.com

# apply resource
k apply -f artifacts/buyers.yaml -f artifacts/sellers.yaml

k get sellers
k get buyers
```

```sh
k get buyers -o jsonpath='{range .items[*]}{@.metadata.name}:{@.spec.name}{"\n"}{end}'
k get sellers -o jsonpath='{range .items[*]}{@.metadata.name}:{@.spec.name}{"\n"}{end}'
```

### Step 2 - 创建 Controller

> 为了便于项目的管理，需要尽量保持 项目、module、api group 等名称保持一定的统一

- 初始化项目

```sh
go mod init github.com/Anddd7/playground-k8s-crd
```

- 创建 pkg/apis/playgroundk8scrd/register.go

声明能够被 client-go 识别的参数和变量（如果用其它的语言 sdk，也需要 follow 相应的规则）

```sh
mkdir -p pkg/apis/playgroundk8scrd
touch pkg/apis/playgroundk8scrd/register.go
```

```go
package playgroundk8scrd

const (
 // same as the group name of the crd 'spec.group'
 GroupName = "playgroundk8scrd.anddd7.github.com"
 Version   = "v1alpha1"
)
```

- 创建 skeleton 代码
  - pkg/apis/playgroundk8scrd/v1alpha1/doc.go
  - pkg/apis/playgroundk8scrd/v1alpha1/types.go
  - pkg/apis/playgroundk8scrd/v1alpha1/register.go

参照 kubernetes/sample-controller 定义 CRD 资源在 Go 代码中的结构体


- 目录结构

```sh
$ tree                                         
.
├── LICENSE
├── README.md
├── artifacts
│   ├── buyers.yaml
│   ├── crd-buyer.yaml
│   ├── crd-seller.yaml
│   └── sellers.yaml
├── go.mod
├── go.sum
└── pkg
    └── apis
        └── playgroundk8scrd
            ├── register.go
            └── v1alpha1
                ├── doc.go
                ├── register.go
                └── types.go
```

### Step 3 - 编写 Controller

- 准备 code generato，从 sample-controller 里复制 hack 文件夹

- 修改 hack/update-codegen.sh (项目名和包名)，并执行

```sh
# ... resolve compile issue
go mod vendor
chmod -R 777 vendor
./hack/update-codegen.sh
```

### 推荐阅读

- [Kubernetes CRD 详解（Custom Resource Definition）](https://mp.weixin.qq.com/s?__biz=MzIzNzU5NTYzMA==&mid=2247512881&idx=1&sn=e5595b6d101432112d498ffd7cbe5901&chksm=e8c4cdb0dfb344a620aa10bcc283212a00e075e0b3db60e43cf87f03f9832b8d1d6733a8b16f&scene=178&cur_album_id=1990567114293739521#rd)
- <https://github.com/kubernetes/sample-controller>

## Operator

### Step 1

### Step 2

### Step 3

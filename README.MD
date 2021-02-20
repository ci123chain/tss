## Go语言Web框架基线版本

### 一.框架简介

此web框架已gin为核心，包含了mysq、redis、rabbitmq、log、es等基础组件。
此框架内dockerfile和makefile已适配中台的打镜像流程，一般情况下不需要对这两个文件进行修改。

~~~tree
├── controller 	//控制器
│   └── v1
├── dao 				//dao层
│   ├── mysql
│   └── redis
├── db 					//mysql文件初始化
├── docs 				//swagger 接口文档
├── middleware 	//中间件
├── model 			//model层
│   ├── mapi
│   ├── mdb
│   ├── mmysql
│   └── mparam
├── pconst 			//常量定义
├── route 			//路由层
│   └── v1
├── script 			//脚本层
│   └── logic
├── server 			//服务启动
├── service 		//业务层
├── tgo 				//基础组件
└── util 				//工具包
    ├── curl
    └── ip
~~~



### 二.框架分层

#### 2.1.路由层

框架采用强路由模式，支持路由群组、中间件模式，自带跨域组件并默认开启，路由文件存放在 route 目录下

#### 2.2.控制器层

控制器存放在controller目录下，并区分版本，此层仅针对参数进行过滤处理，不处理相关业务。已集成swagger自动生成接口文档

#### 2.3.业务层

业务层存放在service目录下，用来处理业务相关的逻辑

#### 2.4.数据处理层

业务层存放在dao目录下，并根据下游不同服务放入不同目录下，此层仅处理数据，不处理业务相关逻辑

#### 2.5.对象层

对象层存放在mode目录下，并根据对象的不同类型存放下级不同目录

#### 2.6.中间件层

中间件层存放在middleware目录下，用以对同一分组或具有相同特征的路由进行全局业务处理

#### 2.7.脚本层

脚本层存放在script目录下，用以处理非http类型的业务类型

#### 2.8.常量层

常量定义存放在pconst目录下

### 三.整体流程

框架整体请求请求流程为route - middleware - controller - service - dao，控制器、业务、数据处理禁止逆向调用

### 四.请求示例

参考控制器下 log_platform.go 文件


有任何疑问或者建议，欢迎随时联系基础架构部：陶圣

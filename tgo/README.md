# tgo


#### 模块
~~~~
config:配置
util:工具
dao:数据访问
model:数据模型
gin配置路由

~~~~



#### 生命周期说明

~~~~

启动

0、扫描配置检测格式并配置写进内存
1、针对配置进行初始化操作：db连接池、grpc连接池、cache连接池或其他服务检测可用性
2、启动http或grpc服务

关闭
0、信号通知进程关闭，进入关闭流程http、grpc走shutdown流程
1、shutdown中针对配置之前初始化的服务进行主动close操作

~~~~



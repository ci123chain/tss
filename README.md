该库为tss-lib门限签名算法加上p2p功能的密码机算法，未封装http接口以及上层应用。

注：当前版本暂未实现密钥重组。

##使用流程：
1.生成p2p节点config文件。 （已生成好的模版在test1、test2、test3文件夹，如再次生成用来本地多节点测试需要手动修改config.toml中的相应port）
    
2.启动p2p节点。    

3.等待节点连接后调用node.keygen方法。传入sessionID（密码唯一区分id）和门限数threshold（大于等于1/2节点数，小于节点数）。      

4.keygen done完成后，调用node.signing方法，传入和初始化相同的sessionID和需签名的数据msg。 

5.获取到signature后进行verify，验证通过则签名完成。(当前未存储signature，如业务需要可通过channel在signing完毕时取出，进行存储)

##具体使用
见node/node_test.go
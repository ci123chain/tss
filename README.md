该库为tss-lib门限签名算法加上p2p功能的密码机算法，未封装http接口以及上层应用。

注：当前版本暂未实现密钥重组。

使用流程：

1.生成节点config文件。 
    
2.启动节点。    

3.等待节点连接后调用node.keygen方法。传入sessionID和门限数threshold（大于等于1/2节点数）。      

4.keygen done完成后，调用node.signing方法，传入sessionID和需签名的数据msg。 

5.获取到signature后进行verify，验证通过则签名完成。

具体调用见node/node_test.go
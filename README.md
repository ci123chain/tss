该库为tss-lib门限签名算法加上p2p功能的密码机算法，未封装http接口以及上层应用。

注：当前版本暂未实现密钥重组。

## 使用流程：
1.生成p2p节点config文件。 （已生成好的模版在test1、test2、test3文件夹，如再次生成用来本地多节点测试需要手动修改config.toml中的相应port）

```	
privKey1, _, address1, _ := validator.NewValidatorKey()         
privKey2, _, address2, _ := validator.NewValidatorKey()
privKey3, _, address3, _ := validator.NewValidatorKey()

persistentPeer := strings.ToLower(address1) + "@127.0.0.1:26656" + "," + strings.ToLower(address2) + "@127.0.0.1:36656" + "," + strings.ToLower(address3) + "@127.0.0.1:46656"

initFiles1, _ := create.NewInitFiles(privKey1, persistentPeer, false)
writeConfigFile(root1, initFiles1)

initFiles2, _ := create.NewInitFiles(privKey2, persistentPeer, false)
writeConfigFile(root2, initFiles2)

initFiles3, _ := create.NewInitFiles(privKey3, persistentPeer, false)
writeConfigFile(root3, initFiles3)
```


2.启动p2p节点。    
```
cfg, err := getConfig(root)
if err != nil {
	panic(err)
}

cfg.SetRoot(root)

// create node
n, err := DefaultNewNode(cfg, log.TestingLogger())
if err != nil {
	panic(err)
}

err := n.Start()
```

3.等待节点连接后调用node.keygen方法。传入sessionID（密码唯一区分id）和门限数threshold（大于等于1/2节点数，小于节点数）。      
```
//传入门限数和sessionID
resCh := n.Keygen(2, sessionID)
if resCh == nil {
	return
}

select {
case <-resCh:
	//wait for n2, n3 done, just for test
	time.Sleep(2 * time.Second)
}
```

4.keygen done完成后，调用node.signing方法，传入和初始化相同的sessionID和需签名的数据msg。 
```
sessionID := threshold.SessionID("session-1")
msg := big.NewInt(42)
resCh, err := n.Signing(msg, sessionID)
require.NoError(t, err)
```

5.获取到signature后进行verify，验证通过则签名完成。
```
select {
case signature := <-resCh:
	err := n.Verify(msg, sessionID, signature)
	require.NoError(t, err)
}
```

## 具体使用
见node/node_test.go
# euphonyqr_token_backend

1. 准备</br>
  请和动听工作人员联系获取售前服务文档，并全部完成。如果只是想尝试一下SDK，可以跳过这一步。</br>
  开发者需要从动听官网(http://www.euphonyqr.com) 或是开发服(http://sandbox.euphonyqr.com) 申请帐号，用APPKEY和SECKEY来获得token</br>
2. 实现后台接口</br>
 此服务器实现了返回token的方法，具体地址可在SDK初始化中定义。此方法接受两个参数appkey和refresh，返回缓存的token。</br>
  此项目仅为参考实现，不建议使用在正式服务器中，动听不对此项目做任何保证。你可以用任何熟悉的语言框架来实现相同的业务功能，可以集成布署也可以独立布署。</br> 
  参照wx sdk中index.js中的配置</br></br>
  onLoad：</br>
  detector.init({
      appKey:"121e87d73077403eadd9ab4fec2d9973",//请替换成自己的APPKEY</br>
      abortTimeout: 3000,</br>
      detectTimeout: 5000,</br>
      debugLog: true,</br>
    });</br></br>
3. 测试</br>
  用苹果电脑或是独立电箱播放testsound目录中的mp3音乐，在手机上测试识别结果
4. 注意事项和常见问题：</br>
  1）请查看一下main.go中的注释，目前只提供go语言的参考实现，可以参考main.go中的注释来实现其它语言的实现。</br>
  2）func main()中定义的环境变量只在这个DEMO中使用，你的实现代码可以完全无视这些变量。</br>
  3）请分清楚APPKEY和SECKEY是在正式服 www.euphonyqr.com申请的还是在测试服sandbox.euphonyqr.com 申请的。线下店帐号和APP帐号都要在同一平台上申请才能互相操作。在调用https://api.euphonyqr.com/api/buyfulltoken 时要记得传入正确的sandbox值。</br>
  4）请确保网络通畅并且可以连接外网。</br>
  5）buyfulltoken会不定时过期，微信SDK发现buyfulltoken过期后会发出带有refresh的请求，这时服务器可以请求更新buyfulltoken并返回给微信SDK。buyfull token无需长期保存，没有过期时间策略，只要在内存中记录服务器返回值既可。每次请求https://api.euphonyqr.com/api/buyfulltoken 时在oldtoken参数中传入内存中记录的buyfull token既可，如果没有就传空字符串。</br>
  6) 建议检查WXAPPID以免后台服务被别的小程序冒用。</br>
  7) 请至少在APP帐号下购买一个渠道后再进行测试，并且请在渠道中自行设定，自行设定，自行设定（重要的事情说三遍）识别结果，可以为任何字符串包括JSON。</br>
  8) 请在调用https://api.euphonyqr.com/api/buyfulltoken 时加入多线程或多进程锁，以确保同时只有一个请求会更新buyfulltoken，否则过多的请求引起动听服务器将你的服务暂时限制一段时间。</br></br>



有疑问请联系QQ:55489181

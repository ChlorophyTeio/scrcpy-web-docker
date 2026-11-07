const https=require('https'),http=require('http'),fs=require('fs');
const server=https.createServer({key:fs.readFileSync(process.env.SCRCPYCAT_TLS_KEY||'/data/tls/server.key'),cert:fs.readFileSync(process.env.SCRCPYCAT_TLS_CERT||'/data/tls/server.crt')},(req,res)=>{
 const upstream=http.request({hostname:'127.0.0.1',port:8443,path:req.url,method:req.method,headers:{...req.headers,'x-forwarded-proto':'https'}},r=>{res.writeHead(r.statusCode,r.headers);r.pipe(res)});
 upstream.on('error',()=>{res.writeHead(502);res.end('Service starting')});req.pipe(upstream);
});
server.on('upgrade',(req,socket,head)=>{
 const upstream=http.request({hostname:'127.0.0.1',port:8443,path:req.url,method:req.method,headers:{...req.headers,'x-forwarded-proto':'https'}});
 upstream.on('upgrade',(res,remote,extra)=>{socket.write('HTTP/1.1 101 Switching Protocols\r\n'+Object.entries(res.headers).map(([k,v])=>k+': '+v).join('\r\n')+'\r\n\r\n');if(extra.length)socket.write(extra);if(head.length)remote.write(head);socket.pipe(remote);remote.pipe(socket);remote.on('error',()=>socket.destroy());socket.on('error',()=>remote.destroy())});
 upstream.on('response',r=>{socket.end('HTTP/1.1 '+r.statusCode+' Rejected\r\n\r\n')});upstream.on('error',()=>socket.destroy());upstream.end();
});server.listen(9443,'0.0.0.0');

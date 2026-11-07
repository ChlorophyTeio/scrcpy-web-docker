import {defineStore} from 'pinia'
import {ref,computed} from 'vue'
import {useAuthStore} from './auth'
export const usePluginStore=defineStore('plugins',()=>{
 const entries=ref([]),loaded=ref(false),auth=useAuthStore()
 let timer=null,inFlight=null,generation=0
 const available=computed(()=>entries.value.filter(p=>p.available))
 function isAvailable(id){return available.value.some(p=>p.id===id)}
 async function refresh(){
  if(!auth.isLoggedIn||!auth.isAdmin){stop();return}
  if(inFlight)return inFlight
  const run=generation
  inFlight=(async()=>{try{
   const r=await fetch('/api/plugins',{headers:{Authorization:`Bearer ${auth.token}`},signal:AbortSignal.timeout(5000)})
   if(!r.ok)throw new Error('Plugin discovery failed')
   const v=await r.json();if(run===generation){entries.value=Array.isArray(v.plugins)?v.plugins:[];loaded.value=true}
  }catch{if(run===generation){entries.value=[];loaded.value=true}}finally{inFlight=null}})()
  return inFlight
 }
 function visible(){if(!document.hidden)refresh()}
 function start(){if(timer)return;refresh();timer=setInterval(refresh,10000);document.addEventListener('visibilitychange',visible)}
 function stop(){generation++;entries.value=[];loaded.value=false;if(timer)clearInterval(timer);timer=null;document.removeEventListener('visibilitychange',visible)}
 return {entries,available,loaded,isAvailable,refresh,start,stop}
})

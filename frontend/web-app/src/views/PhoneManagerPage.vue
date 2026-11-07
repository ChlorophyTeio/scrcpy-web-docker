<template><section class="phone-manager-page">
 <header class="phone-manager-header"><div><h2>手机管理</h2><p>文件、进程与应用</p></div><label>当前设备<select v-model="selected" aria-label="管理设备"><option value="" disabled>选择在线设备</option><option v-for="device in devices.onlineDevices" :key="device.id" :value="device.id">{{device.name||device.id}} · {{device.info?.model||'Android'}}</option></select></label></header>
 <nav class="phone-manager-tabs" aria-label="手机管理功能"><button v-for="tab in tabs" :key="tab.id" :class="{active:active===tab.id}" @click="changeTab(tab.id)">{{tab.label}}</button></nav>
 <div v-if="!selected" class="phone-manager-empty">暂无在线设备。请先连接手机并确认 USB 调试授权。</div>
 <FileManagerPage v-if="auth.canFileLibrary&&selected" v-show="active==='files'" :device-id="selected" embedded/>
 <DeviceOperationsPanel v-if="active!=='files'&&selected&&auth.canShell" :device-id="selected" :mode="active" @files="changeTab('files')"/>
</section></template>
<script setup>
import {ref,computed,watch} from 'vue'
import {useRoute,useRouter} from 'vue-router'
import {useDeviceStore} from '@/stores/devices'
import {useAuthStore} from '@/stores/auth'
import FileManagerPage from './FileManagerPage.vue'
import DeviceOperationsPanel from '@/components/DeviceOperationsPanel.vue'
const devices=useDeviceStore(),auth=useAuthStore(),route=useRoute(),router=useRouter(),selected=ref(''),active=ref('files')
const tabs=computed(()=>[{id:'files',label:'文件管理',allowed:auth.canFileLibrary},{id:'processes',label:'进程管理',allowed:auth.canShell},{id:'apps',label:'应用管理',allowed:auth.canShell}].filter(t=>t.allowed))
watch(()=>[route.query.device,devices.onlineDevices.map(d=>d.id).join(',')],()=>{const ids=devices.onlineDevices.map(d=>d.id);const requested=String(route.query.device||'');if(ids.includes(requested))selected.value=requested;else if(!ids.includes(selected.value))selected.value=ids.includes(devices.activeDeviceId)?devices.activeDeviceId:ids[0]||''},{immediate:true})
watch(()=>[route.query.tab,tabs.value.map(t=>t.id).join(',')],()=>{active.value=tabs.value.some(t=>t.id===route.query.tab)?route.query.tab:tabs.value[0]?.id||'files'},{immediate:true})
watch(selected,value=>{if(value)router.replace({path:'/files',query:{...route.query,device:value,tab:active.value}})})
function changeTab(id){active.value=id;router.replace({path:'/files',query:{...route.query,device:selected.value,tab:id}})}
</script>

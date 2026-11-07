<template>
 <component v-if="plugin?.available&&view" :is="view" :key="plugin.id"/>
 <section v-else class="plugin-service-state" role="status"><h2>{{plugin?.name||'插件'}}</h2><p>{{!plugins.loaded?'正在检查插件服务…':!plugin?.available?'插件服务未启动，入口将在服务就绪后自动恢复。':'插件服务已连接，请注册对应的界面组件。'}}</p><button @click="plugins.refresh">重新检查</button></section>
</template>
<script setup>
import {computed,onMounted} from 'vue'
import {useRoute} from 'vue-router'
import {usePluginStore} from '@/stores/plugins'
import {pluginViews} from '@/plugins/registry'
const route=useRoute(),plugins=usePluginStore()
const plugin=computed(()=>plugins.entries.find(p=>p.id===route.params.pluginId))
const view=computed(()=>pluginViews[plugin.value?.frontend])
onMounted(()=>plugins.refresh())
</script>
<style scoped>.plugin-service-state{padding:28px;color:#c8d8e8}.plugin-service-state p{margin:16px 0;line-height:1.7}.plugin-service-state button{padding:9px 16px;border:1px solid #3e5c78;border-radius:7px;background:#1b2c40;color:#9dd8ff}</style>

<template><details class="maa-raw-parameters"><summary>核心参数 JSON</summary><p class="maa-section-note">仅编辑当前任务的参数对象。应用后需保存或检查配置；文件路径请使用网页上传入口。</p><textarea v-model="draft" spellcheck="false" :disabled="disabled" aria-label="核心参数 JSON 编辑器"/><p v-if="error" role="alert" class="maa-import-error">{{error}}</p><button @click="apply" :disabled="disabled">应用核心参数</button></details></template>
<script setup>
import {ref,watch} from 'vue'
const props=defineProps({model:{type:Object,required:true},disabled:Boolean})
const emit=defineEmits(['apply'])
const draft=ref(''),error=ref('')
watch(()=>props.model,value=>{draft.value=JSON.stringify(value,null,2);error.value=''},{immediate:true})
function apply(){try{if(draft.value.length>100000)throw new Error('参数不能超过 100 KB');const value=JSON.parse(draft.value);if(!value||typeof value!=='object'||Array.isArray(value))throw new Error('参数必须为 JSON 对象');emit('apply',value);error.value=''}catch(e){error.value=e instanceof SyntaxError?'JSON 格式有误，请检查括号、引号和逗号':e.message}}
</script>

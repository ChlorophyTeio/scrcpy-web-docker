<template>
 <div class="maa-fields">
  <template v-for="field in fields" :key="field[0]">
   <label v-if="field[2]==='bool'" class="maa-check-field"><input type="checkbox" v-model="model[field[0]]" :disabled="disabled"/><span>{{field[1]}}</span></label>
   <div v-else-if="field[2]==='multi'" class="maa-multifield"><span>{{field[1]}}</span><div><label v-for="choice in field[3]" :key="choice[0]"><input type="checkbox" :value="choice[0]" v-model="model[field[0]]" :disabled="disabled"/>{{choice[1]}}</label></div></div>
   <label v-else class="maa-value-field"><span>{{field[1]}}</span>
    <select v-if="field[2]==='select'" v-model="model[field[0]]" :disabled="disabled"><option v-for="choice in field[3]" :value="choice[0]" :key="choice[0]">{{choice[1]}}</option></select>
    <input v-else-if="field[2]==='number'" type="number" v-model.number="model[field[0]]" :min="field[3]" :max="field[4]" :step="field[5]||1" :disabled="disabled"/>
    <input v-else-if="field[2]==='list'" :value="(model[field[0]]||[]).join(', ')" @change="model[field[0]]=$event.target.value.split(/[,，]/).map(s=>s.trim()).filter(Boolean)" :disabled="disabled"/>
    <input v-else type="text" v-model="model[field[0]]" :disabled="disabled" :autocapitalize="field[0]==='package'?'none':undefined" :spellcheck="field[0]==='package'?false:undefined" :placeholder="field[0]==='package'?'com.hypergryph.arknights':undefined"/>
   </label>
  </template>
 </div>
</template>
<script setup>
defineProps({fields:{type:Array,default:()=>[]},model:{type:Object,required:true},disabled:Boolean})
</script>

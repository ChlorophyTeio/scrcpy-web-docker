import {defineAsyncComponent} from 'vue'
// Service discovery is data-driven; feature-specific views are compiled modules.
// Add a view here plus a service manifest to integrate another plugin.
export const pluginViews={maa:defineAsyncComponent(()=>import('./maa/MaaPage.vue'))}

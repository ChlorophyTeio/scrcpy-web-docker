import test from 'node:test'
import assert from 'node:assert/strict'
import {fightLimit,selectedPlan} from './maaTasks.js'
test('unchecked limits do not consume resources and unlimited runs use core default',()=>{const p={};fightLimit(p,'medicine',false);fightLimit(p,'stone',false);fightLimit(p,'times',false);assert.deepEqual(p,{medicine:0,stone:0,times:2147483647});fightLimit(p,'medicine',true);assert.equal(p.medicine,1)})
test('task checkbox click order does not reorder execution or mutate saved config',()=>{const config={enabled:['fight','startup'],params:{fight:{stage:'1-7'}}};const plan=selectedPlan(config,['fight','startup','update']);assert.deepEqual(plan.enabled,['startup','fight','update']);plan.params.fight.stage='CE-6';assert.equal(config.params.fight.stage,'1-7');assert.deepEqual(config.enabled,['fight','startup'])})

test('launch is first and user data update is last even with a saved custom order',()=>{const c={order:['update','fight','launch','startup'],params:{}};assert.deepEqual(selectedPlan(c,['update','fight','startup','launch']).enabled,['launch','fight','startup','update'])})

import test from 'node:test'
import assert from 'node:assert/strict'
import {normalizeStageCode,stageVersionOlder} from './maaStages.js'
test('stage input normalization preserves core navigation names',()=>{for(const [a,b] of [[' ce6 ','CE-6'],['1‑7','1-7'],['annihilation','Annihilation'],['ssreopen-at','SSReopen-AT'],['15-3-hard','15-3-HARD']])assert.equal(normalizeStageCode(a),b)})
test('activity minimum core uses semantic version comparison',()=>{assert.equal(stageVersionOlder('MaaCore v6.20.1','v6.21.1'),true);assert.equal(stageVersionOlder('MaaCore v6.21.1','v6.21.1'),false);assert.equal(stageVersionOlder('unknown','v6.21.1'),false)})

import test from 'node:test'
import assert from 'node:assert/strict'
import {normalizeDevicePath,validFileName} from './deviceManagement.js'
test('file navigation normalizes dot segments without decoding filenames',()=>{assert.equal(normalizeDevicePath('/sdcard/a/../Download//'),' /sdcard/Download'.trim());assert.equal(normalizeDevicePath('/../../'),' /'.trim());assert.equal(normalizeDevicePath('/sdcard/%2F/a'),' /sdcard/%2F/a'.trim())})
test('rename accepts ordinary Unicode names but rejects traversal and separators',()=>{for(const n of ['测试.txt',"a 'quoted';.txt",'.hidden'])assert.equal(validFileName(n),true);for(const n of ['','..','.','../a','a/b','a\\b','a\0'])assert.equal(validFileName(n),false)})

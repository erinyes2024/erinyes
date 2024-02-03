"use strict";
// @ts-nocheck
const express = require("express");
const { createHook, executionAsyncId } = require('async_hooks');
const { trigger } = require('./trigger');
var lastRequest = -1; // request uuid
const headerCoro = new Map();
createHook({
    init(asyncId, type, triggerAsyncId, resource) {
        if (headerCoro.has(triggerAsyncId)) {
            headerCoro.set(asyncId, headerCoro.get(triggerAsyncId));
        }
    },
    before(asyncId) {
        var data = headerCoro.get(asyncId);
        if (typeof (data) != 'undefined' && data != lastRequest) {
            trigger(`flag_data is ${data}`);
            lastRequest = data;
        }
    },
    destroy(asyncId) {
        if (headerCoro.has(asyncId)) {
            headerCoro.delete(asyncId);
        }
    },
    promiseResolve(asyncId) {
        var data = headerCoro.get(asyncId);
        if (typeof (data) != 'undefined' && data != lastRequest) {
            trigger(`flag_data is ${data}`);
            lastRequest = data;
        }
    }
}).enable();
function new_express() {
    const app = express();
    app.use((req, res, next) => {
        if ('uuid' in req.headers) {
            headerCoro.set(executionAsyncId(), req.headers['uuid']);
            // console.log(`uuid: ${req.headers['uuid']}`)
            next();
        }
        else {
            res.status(400).send('Hacker!');
        }
    });
    app.use((req, res, next) => {
        res.on('finish', () => {
            // var data = headerCoro.get(executionAsyncId());
            // if (typeof (data) != 'undefined') {
            //     trigger(`end_flag_data is ${data}`);
            // }
            trigger(`end_flag_data is ${req.headers['uuid']}`);
        });
        next();
    });
    return app;
}
function get_uuid() {
    var data = headerCoro.get(executionAsyncId());
    if (typeof (data) != 'undefined') {
        return data;
    }
}
Object.assign(new_express, express);
exports.new_express = new_express;
exports.get_uuid = get_uuid;

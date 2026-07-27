// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// jest-environment-jsdom (v16) lacks Web Streams; undici (pulled transitively) needs them.
import {ReadableStream, TransformStream, WritableStream} from 'stream/web';

Object.assign(globalThis, {ReadableStream, TransformStream, WritableStream});

export {};

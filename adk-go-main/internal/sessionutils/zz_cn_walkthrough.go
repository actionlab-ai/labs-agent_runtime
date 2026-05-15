// Copyright 2026 Archinfra / yuanyp8
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package sessionutils

// 代码走读：internal/sessionutils 处理状态作用域拆分与合并。
//
// 业务理解：
// 1. EventActions.StateDelta 里可能包含 app:/user:/temp: 等前缀。
// 2. SessionService 追加事件时会把这些 delta 拆到不同作用域。
// 3. 读取 Session 时再把 appState + userState + sessionState 合并成当前可见状态。
//
// 技术设计模式：
// - Scoped State：用 key prefix 实现多层级状态，而不是引入多套 API。

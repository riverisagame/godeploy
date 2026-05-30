# 架构决策记录 (ADR) - 解决 Diff 弹窗文件选择面板不显示的问题

## 1. 问题分析与定位
大项目在触发“预览 Diff”弹窗或“部署”弹窗，或者查看历史任务的 Diff 时，弹窗内的左侧文件树 `.diff-left-sidebar` 处于空数据状态，用户无法进行可选文件过滤。

**根本原因**：
- 在 `Dashboard.vue` 的 `watch` 预加载逻辑之外，当用户直接点击“预览 Diff”或历史“任务 Diff”按钮触发 `previewDeployDiff` 和 `showDiff` 函数时，这两个函数仅更新了扁平结构 `parsedDiffFiles`，并未给 `fileTreeData`、`rawFilesList` 以及 `defaultCheckedKeys` 赋值。
- 这导致在弹窗展示时，绑定的 `:data="fileTreeData"` 为空，使得文件选择树始终不可见或没有可选文件。
- `executeDeploy` 依赖 `rawFilesList` 获取所有可能变更的文件以计算排除列表，由于未被赋值，导致排除列表计算失效。

## 2. 解决方案建议
- **方案**：在 `previewDeployDiff` 和 `showDiff` 触发时，同步根据获取到的文件列表重新调用 `buildTree(files)` 并将结果赋给 `fileTreeData`，同时将原始列表存入 `rawFilesList` 并赋予 `defaultCheckedKeys` 为默认全部选中。
- **性能对冲**：文件树为本地同步解析（通过 `buildTree` 将平铺的 `string[]` 路径转换成树节点），耗时小于 1ms，且每次打开弹窗仅执行一次，无并发和渲染卡顿风险。
- **副作用评估**：仅更新局部的数据绑定，对现有的部署核心流程完全“零侵入”，无负面副作用。

## 3. 影响评估与回退机制
- **影响范围**：仅影响 `Dashboard.vue` 中的 Diff 弹窗数据呈现，不影响后端核心发布和 SQLite 数据存储。
- **回退机制**：如果新逻辑有任何问题，只需撤销 `Dashboard.vue` 内部 `previewDeployDiff` 和 `showDiff` 的改动即可。

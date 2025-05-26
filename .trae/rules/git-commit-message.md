---
alwaysApply: true
scene: git_message
---

请根据 git 暂存区内容生成 commit message，要求：

1. 第一行为简短总结，格式：✨ feat: <中文描述>  或  🐛 fix: <中文描述>  或其他类型前缀
2. 大类前缀使用：✨ feat / 🐛 fix / 🔧 chore / 🧹 refactor / 📦 deps / 🗑️ remove / 🔄 sync / 📝 docs
3. 下面每条改动用小项列出，以 \`- <emoji>\` 开头，中文描述 + 括号内注明涉及文件名
4. 末尾附一个极简一行版本
5. 仅输出 commit message，不要额外解释
6. 常用 emoji 对照表 （按需挑选）：✨ 新功能 feat  🐛 修复 bug  🔧 基础设施/配置 chore  🧹 重构 refactor  📦 依赖更新 deps  🗑️ 删除/清理  🔄 同步/联动逻辑  🏗️ 新建服务/模块  🌐 国际化 i18n  🚀 性能优化  📝 文档/注释  💄 UI/样式  ✅ 测试  🛡️ 安全相关

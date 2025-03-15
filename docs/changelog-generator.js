const fs = require('fs');
const { marked } = require('marked');
const path = require('path');

// 自定义渲染器
const renderer = {
    heading(text, level) {
        const escapedText = text.toLowerCase().replace(/[^\w]+/g, '-');
        
        if (level === 1) {
            return `<h1 class="text-4xl font-bold text-gray-900 mb-8">${text}</h1>`;
        } else if (level === 2) {
            return `
                <div class="mb-8">
                    <h2 class="text-2xl font-semibold text-gray-900 mb-4" id="${escapedText}">
                        ${text}
                    </h2>
                </div>
            `;
        } else if (level === 3) {
            return `<h3 class="text-xl font-medium text-gray-800 mt-6 mb-4">${text}</h3>`;
        }
        return `<h${level} class="font-medium text-gray-800 mt-4 mb-2">${text}</h${level}>`;
    },

    list(body, ordered) {
        const type = ordered ? 'ol' : 'ul';
        return `<${type} class="list-disc list-inside space-y-2 text-gray-600 mb-6 ml-4">${body}</${type}>`;
    },

    listitem(text) {
        return `<li class="text-base">${text}</li>`;
    }
};

// 设置渲染选项
marked.use({ renderer });

// 读取 CHANGELOG.md
const changelogPath = path.join(__dirname, '..', 'CHANGELOG.md');
const outputPath = path.join(__dirname, 'changelog.html');

try {
    const markdown = fs.readFileSync(changelogPath, 'utf8');
    const html = marked(markdown);
    
    // 添加样式包装
    const styledHtml = `
        <div class="changelog-container prose prose-lg max-w-none">
            ${html}
        </div>
    `;
    
    fs.writeFileSync(outputPath, styledHtml);
    console.log('Changelog HTML 生成成功！');
} catch (error) {
    console.error('生成 Changelog HTML 时出错：', error);
} 
const translations = {
    en: {
        "title": "Domain Radar - Login",
        "heading": "📡 Domain Radar",
        "subheading": "Your insight into domain trends.",
        "welcome": "Welcome Back",
        "username_label": "Username",
        "username_placeholder": "Enter your username",
        "password_label": "Password",
        "password_placeholder": "Enter your password",
        "login_btn": "Login",
        "register_btn": "Register",
        "footer": "© 2024 Domain Radar. All rights reserved.",
        "msg_enter_both": "Please enter both username and password",
        "msg_login_failed": "Login failed",
        "msg_register_success": "Registered successfully! You can now login.",
        "msg_register_failed": "Register failed",
        "msg_network_error": "Network error",
        "dash_title": "Domain Radar - Dashboard",
        "nav_brand": "📡 Domain Radar",
        "logout": "Logout",
        "reports_title": "Reports Dashboard",
        "refresh_btn": "Refresh",
        "loading": "Loading reports...",
        "no_reports": "No reports found.",
        "failed_load": "Failed to load reports.",
        "report_contains": "Contains {count} domains",
        "view_details": "View Details →",
        "report_page_title": "Domain Radar | Report Details",
        "loading_report": "Loading report details...",
        "no_id": "No report ID specified",
        "load_error": "Failed to load report",
        "report_header": "📡 Domain Radar Daily Report",
        "deep_analysis_title": "🧠 Global Deep Analysis",
        "macro_trends": "🔍 Macro Trends",
        "opportunities": "🚀 Opportunities",
        "risks": "🛡️ Risks",
        "action_guides": "💡 Action Guides",
        "domain_overview": "📝 Overview",
        "domain_trends": "📈 Trends",
        "key_events": "🔥 Key Events",
        "references": "🔗 References",
        "heat_score": "Heat: {score}/10",
        "switch_lang": "中文",
        "date_cover": "{date} • Covering {count} domains"
    },
    zh: {
        "title": "领域雷达 - 登录",
        "heading": "📡 领域雷达",
        "subheading": "洞察领域趋势，把握未来先机。",
        "welcome": "欢迎回来",
        "username_label": "用户名",
        "username_placeholder": "请输入用户名",
        "password_label": "密码",
        "password_placeholder": "请输入密码",
        "login_btn": "登录",
        "register_btn": "注册",
        "footer": "© 2024 领域雷达. 保留所有权利.",
        "msg_enter_both": "请输入用户名和密码",
        "msg_login_failed": "登录失败",
        "msg_register_success": "注册成功！现在可以登录了。",
        "msg_register_failed": "注册失败",
        "msg_network_error": "网络错误",
        "dash_title": "领域雷达 - 仪表盘",
        "nav_brand": "📡 领域雷达",
        "logout": "退出登录",
        "reports_title": "报表仪表盘",
        "refresh_btn": "刷新",
        "loading": "正在加载报表...",
        "no_reports": "暂无报表。",
        "failed_load": "加载报表失败。",
        "report_contains": "包含 {count} 个领域",
        "view_details": "查看详情 →",
        "report_page_title": "领域雷达 | 报表详情",
        "loading_report": "正在加载报表详情...",
        "no_id": "未指定报表 ID",
        "load_error": "加载报表失败",
        "report_header": "📡 领域雷达日报",
        "deep_analysis_title": "🧠 全局深度解读",
        "macro_trends": "🔍 宏观趋势",
        "opportunities": "🚀 机遇",
        "risks": "🛡️ 风险",
        "action_guides": "💡 行动指南",
        "domain_overview": "📝 综述",
        "domain_trends": "📈 趋势",
        "key_events": "🔥 关键事件",
        "references": "🔗 参考来源",
        "heat_score": "热度: {score}/10",
        "switch_lang": "English",
        "date_cover": "{date} • 覆盖 {count} 个领域"
    }
};

let currentLang = localStorage.getItem('lang') || 'zh'; // Default to Chinese as per user locale hint, or user preference

function setLanguage(lang) {
    currentLang = lang;
    localStorage.setItem('lang', lang);
    updatePage();
}

function toggleLanguage() {
    setLanguage(currentLang === 'en' ? 'zh' : 'en');
}

function t(key, params = {}) {
    let val = (translations[currentLang] && translations[currentLang][key]) || (translations['en'][key]) || key;
    for (const k in params) {
        val = val.replace(`{${k}}`, params[k]);
    }
    return val;
}

function updatePage() {
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        if (el.tagName === 'INPUT' && el.placeholder) {
             el.placeholder = t(key);
        } else {
             el.innerText = t(key);
        }
    });
    
    // Update language switcher button text if exists
    const switcher = document.getElementById('lang-switcher');
    if (switcher) {
        switcher.innerText = t('switch_lang');
    }

    // Update HTML lang attribute
    document.documentElement.lang = currentLang == 'zh' ? 'zh-CN' : 'en';
}

// Expose functions globally
window.t = t;
window.setLanguage = setLanguage;
window.toggleLanguage = toggleLanguage;
window.updatePage = updatePage;
window.currentLang = currentLang;

document.addEventListener('DOMContentLoaded', updatePage);

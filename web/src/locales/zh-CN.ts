import type { MessageSchema } from '@/locales/schema'

const zhCN = {
  app: { name: 'shadcn-vue-go-template' },
  common: {
    action: {
      add: '新增',
      back: '返回',
      cancel: '取消',
      close: '关闭',
      confirm: '确认',
      create: '创建',
      delete: '删除',
      edit: '编辑',
      menu: '打开菜单',
      save: '保存',
      search: '搜索',
      submit: '提交',
      update: '更新',
    },
    field: { email: '邮箱', password: '密码' },
    feedback: {
      networkError: '网络异常，请稍后重试。',
      required: '此字段为必填项。',
      unknownError: '发生未知错误，请稍后重试。',
    },
    state: { empty: '暂无内容', loading: '加载中...', noData: '暂无数据' },
  },
  route: { dashboard: '仪表盘', login: '登录', notFound: '页面不存在' },
  auth: {
    signIn: {
      description: '使用你的账号凭据继续。',
      emailPlaceholder: '请输入你的电子邮箱地址',
      forgotPassword: '忘记密码？',
      loginFailed: '登录失败，请重试。',
      passwordPlaceholder: '请输入你的密码',
      rememberMe: '记住我',
      signingIn: '登录中...',
      submit: '登录',
      title: '欢迎回来',
    },
  },
  table: {
    action: {
      addSection: '添加章节',
      customizeColumns: '自定义列',
      columns: '列',
      delete: '删除',
      edit: '编辑',
      favorite: '收藏',
      makeCopy: '复制',
    },
    column: {
      header: '标题',
      limit: '限制',
      reviewer: '审核员',
      sectionType: '章节类型',
      status: '状态',
      target: '目标',
    },
    empty: '暂无数据',
    pagination: {
      rowsPerPage: '每页行数',
      pageOf: '第 {page} 页，共 {total} 页',
      goToFirstPage: '跳转到第一页',
      goToPreviousPage: '上一页',
      goToNextPage: '下一页',
      goToLastPage: '跳转到最后一页',
      rowSelected: '已选择 {selected} / {total} 行',
    },
    select: { assignReviewer: '分配审核员', view: '选择视图' },
    tab: {
      outline: '大纲',
      pastPerformance: '过往表现',
      keyPersonnel: '关键人员',
      focusDocuments: '重点文档',
    },
  },
  notFound: { description: '您访问的页面不存在。' },
  sonner: { loginSuccess: '登录成功！' },
  nav: {
    main: { dashboard: '仪表盘', lifecycle: '生命周期', analytics: '分析', projects: '项目', team: '团队' },
    documents: { dataLibrary: '数据库', reports: '报告', wordAssistant: 'Word 助手', label: '文档', more: '更多' },
    secondary: { settings: '设置', getHelp: '获取帮助', search: '搜索' },
    user: { account: '账户', billing: '账单', notifications: '通知', language: '语言', switchLanguage: '切换语言', logout: '退出登录' },
  },
} as const satisfies MessageSchema

export default zhCN

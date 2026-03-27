export interface MessageSchema {
  app: { name: string; title: string; titleWithPage: string }
  common: {
    action: {
      add: string
      back: string
      cancel: string
      close: string
      confirm: string
      create: string
      delete: string
      edit: string
      menu: string
      save: string
      search: string
      submit: string
      update: string
    }
    field: { email: string; password: string; username: string; usernameOrEmail: string; name: string; confirmPassword: string }
    feedback: { loadFailed: string; networkError: string; required: string; unknownError: string }
    state: { empty: string; loading: string; noData: string; neverUsed: string }
    userRole: { 0: string; 1: string; 2: string }
  }
  apiError: {
    unknown: string
    invalidCredentials: string
    unauthorized: string
    usernameRequired: string
    usernameTaken: string
    emailTaken: string
    currentPasswordInvalid: string
    passwordTooShort: string
    avatarRequired: string
    avatarInvalidType: string
    avatarTooLarge: string
    profileUpdateFailed: string
    avatarUploadFailed: string
    passwordUpdateFailed: string
    accountDeleteFailed: string
    superAdminDeleteForbidden: string
    systemLogStreamFailed: string
  }
  route: { dashboard: string; login: string; notFound: string; settings: string; register: string; systemLogs: string; tasks: string; feedback: { loadFailed: string } }
  settings: {
    title: string
    description: string
    saved: string
    save: string
    tabs: { basic: string; account: string; notifications: string }
    account: {
      profile: string
      profileDesc: string
      profileUpdated: string
      saveProfile: string
      savingProfile: string
      changeAvatar: string
      avatarHint: string
      avatarUnsupportedType: string
      avatarFileTooLarge: string
      avatarProcessFailed: string
      username: string
      usernameRequired: string
      usernamePlaceholder: string
      email: string
      emailNotSet: string
      emailPlaceholder: string
      edit: string
      editProfile: string
      editProfileDesc: string
      mustChangePasswordTitle: string
      mustChangePasswordDesc: string
      password: string
      passwordDesc: string
      currentPassword: string
      currentPasswordPlaceholder: string
      newPassword: string
      newPasswordPlaceholder: string
      confirmPassword: string
      confirmPasswordPlaceholder: string
      passwordMismatch: string
      passwordUpdated: string
      updatingPassword: string
      updatePassword: string
      dangerZone: string
      dangerZoneDesc: string
      dangerZoneConfirm: string
      deleteAccount: string
      deleteAccountConfirm: string
      deleteAccountSuccess: string
      superAdminDeleteForbidden: string
    }
    basic: {
      theme: string
      themeDesc: string
      colorTheme: string
      light: string
      dark: string
      system: string
      selectTheme: string
      language: string
      selectLanguage: string
      dataRefreshInterval: string
      dataRefreshIntervalDesc: string
    }
    notifications: {
      title: string
      desc: string
      email: string
      emailDesc: string
      push: string
      pushDesc: string
      digest: string
      digestDesc: string
      security: string
      securityDesc: string
    }
  }
  auth: {
    signIn: {
      description: string
      identifierPlaceholder: string
      forgotPassword: string
      loginFailed: string
      loginSuccess: string
      passwordPlaceholder: string
      rememberMe: string
      signingIn: string
      submit: string
      title: string
    }
    signUp: {
      title: string
      description: string
      signIn: string
      usernamePlaceholder: string
      emailPlaceholder: string
      passwordPlaceholder: string
      confirmPassword: string
      confirmPasswordPlaceholder: string
      creating: string
      submit: string
      or: string
      continueWithGithub: string
      termsPrefix: string
      terms: string
      termsAnd: string
      privacy: string
    }
  }
  table: {
    action: {
      addSection: string
      customizeColumns: string
      columns: string
      delete: string
      edit: string
      favorite: string
      makeCopy: string
    }
    column: {
      header: string
      limit: string
      reviewer: string
      sectionType: string
      status: string
      target: string
    }
    empty: string
    pagination: {
      rowsPerPage: string
      pageOf: string
      goToFirstPage: string
      goToPreviousPage: string
      goToNextPage: string
      goToLastPage: string
      rowSelected: string
    }
    select: { assignReviewer: string; view: string }
    tab: { outline: string; pastPerformance: string; keyPersonnel: string; focusDocuments: string }
  }
  notFound: { description: string }
  theme: { light: string; dark: string; system: string }
  systemLogs: {
    title: string
    description: string
    summary: { buffered: string }
    connection: { connected: string; connecting: string; disconnected: string }
    console: { title: string }
    actions: { clear: string; export: string; pauseFollow: string; resumeFollow: string; reconnect: string }
    export: {
      title: string
      description: string
      fields: { count: string; level: string; format: string }
      counts: { ALL: string; 10: string; 20: string; 50: string; 100: string }
      formats: { csv: string; txt: string; json: string }
      preview: string
    }
    filters: {
      searchPlaceholder: string
      levelPlaceholder: string
      level: { all: string; DEBUG: string; INFO: string; WARN: string; ERROR: string }
    }
    empty: { title: string; description: string }
    feedback: { exportSuccess: string; exportEmpty: string }
  }
  nav: {
    main: { dashboard: string; tasks: string; lifecycle: string; analytics: string; projects: string; team: string }
    management: { systemLogs: string; label: string }
    secondary: { settings: string; getHelp: string; search: string }
    user: { account: string; billing: string; notifications: string; language: string; switchLanguage: string; logout: string }
  }
}

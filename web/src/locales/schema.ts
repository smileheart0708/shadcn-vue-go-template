export interface MessageSchema {
  app: { name: string; title: string; titleWithPage: string }
  common: {
    action: {
      add: string
      back: string
      cancel: string
      close: string
      confirm: string
      copy: string
      create: string
      delete: string
      download: string
      edit: string
      export: string
      filter: string
      import: string
      menu: string
      next: string
      refresh: string
      reset: string
      retry: string
      save: string
      search: string
      sort: string
      submit: string
      update: string
      upload: string
      view: string
    }
    field: { email: string; password: string; username: string; usernameOrEmail: string; name: string; confirmPassword: string }
    feedback: { loadFailed: string; networkError: string; required: string; unknownError: string }
    state: {
      completed: string
      disabled: string
      empty: string
      enabled: string
      failed: string
      loading: string
      neverUsed: string
      noData: string
      noResult: string
      pending: string
      processing: string
      success: string
    }
    text: { yes: string; no: string; none: string; all: string; optional: string; required: string }
    role: { owner: string; admin: string; user: string }
  }
  apiError: {
    unknown: string
    invalidCredentials: string
    accountDisabled: string
    unauthorized: string
    usernameRequired: string
    usernameTaken: string
    emailTaken: string
    currentPasswordInvalid: string
    passwordTooShort: string
    registrationDisabled: string
    invalidRegistrationMode: string
    setupRequired: string
    setupCompleted: string
    avatarRequired: string
    avatarInvalidType: string
    avatarTooLarge: string
    profileUpdateFailed: string
    avatarUploadFailed: string
    passwordUpdateFailed: string
    accountDeleteFailed: string
    accountDeleteForbidden: string
    invalidRoleKeys: string
    systemLogStreamFailed: string
  }
  route: {
    systemConfig: string
    adminUsers: string
    dashboard: string
    login: string
    notFound: string
    settings: string
    register: string
    setup: string
    systemLogs: string
    tasks: string
    feedback: { loadFailed: string }
  }
  systemConfig: {
    title: string
    description: string
    badge: string
    updatedAt: string
    registration: {
      title: string
      description: string
      updatedAt: string
      options: {
        disabled: { title: string; description: string }
        password: { title: string; description: string }
      }
    }
    observability: { title: string; description: string; cta: string }
    cards: {
      auth: { title: string; description: string }
      effectivePolicy: { title: string; description: string }
    }
    fields: {
      authMode: { title: string; description: string }
      registrationMode: { title: string; description: string }
      adminUserCreateEnabled: { title: string; description: string }
      selfServiceAccountDeletionEnabled: { title: string; description: string }
      passwordLoginEnabled: { title: string; description: string }
    }
    options: {
      authMode: { singleUser: string; multiUser: string }
      registrationMode: { disabled: string; public: string }
    }
    policy: {
      authMode: string
      registrationMode: string
      publicRegistration: string
      adminUserCreate: string
      selfServiceAccountDeletion: string
    }
    actions: { retry: string }
    feedback: { loadFailedTitle: string; loadFailed: string; saving: string; saved: string; saveFailed: string }
  }
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
      statusActive: string
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
      deleteAccountOwnerForbidden: string
      deleteAccountUnavailable: string
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
  setup: {
    title: string
    creating: string
    success: string
    failed: string
    submit: string
    step1: string
    step2: string
    passwordMismatch: string
    complete: string
  }
  auth: {
    signIn: {
      description: string
      identifierPlaceholder: string
      forgotPassword: string
      loginFailed: string
      loginSuccess: string
      noAccount: string
      passwordPlaceholder: string
      register: string
      rememberMe: string
      signingIn: string
      submit: string
      title: string
    }
    signUp: {
      title: string
      description: string
      disabledTitle: string
      disabledDescription: string
      disabledHint: string
      emailOptional: string
      signIn: string
      usernamePlaceholder: string
      emailPlaceholder: string
      passwordPlaceholder: string
      confirmPassword: string
      confirmPasswordPlaceholder: string
      passwordMismatch: string
      creating: string
      registerSuccess: string
      registerFailed: string
      loadFailedTitle: string
      policyLoadFailed: string
      retry: string
      submit: string
    }
  }
  adminUsers: {
    title: string
    description: string
    badge: string
    actions: {
      createUser: string
      refresh: string
      retry: string
      disable: string
      enable: string
      previousPage: string
      nextPage: string
    }
    filters: {
      title: string
      description: string
      searchPlaceholder: string
      rolePlaceholder: string
      statusPlaceholder: string
      roleAll: string
      statusAll: string
    }
    table: {
      title: string
      summary: string
      username: string
      email: string
      role: string
      status: string
      createdAt: string
      actions: string
      empty: string
      noEmail: string
      pageSummary: string
    }
    status: { active: string; disabled: string }
    dialog: {
      createTitle: string
      createDescription: string
      createSubmit: string
      editTitle: string
      editDescription: string
      editSubmit: string
      usernamePlaceholder: string
      emailPlaceholder: string
      passwordPlaceholder: string
      passwordHint: string
    }
    confirm: {
      disableTitle: string
      disableDescription: string
      enableTitle: string
      enableDescription: string
    }
    feedback: {
      loadFailedTitle: string
      loadFailed: string
      refreshing: string
      creating: string
      createSuccess: string
      createFailed: string
      updating: string
      updateSuccess: string
      updateFailed: string
      disabling: string
      disableSuccess: string
      disableFailed: string
      enabling: string
      enableSuccess: string
      enableFailed: string
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
    tabs: { console: string; audit: string }
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
    audit: {
      title: string
      description: string
      pageSummary: string
      outcome: { success: string; failure: string }
      table: {
        occurredAt: string
        eventType: string
        outcome: string
        actor: string
        subject: string
        reason: string
        empty: string
      }
      feedback: { loadFailedTitle: string; loadFailed: string }
    }
    feedback: { exportSuccess: string; exportEmpty: string }
  }
  nav: {
    main: { dashboard: string; tasks: string; lifecycle: string; analytics: string; projects: string; team: string }
    management: { systemConfig: string; users: string; systemLogs: string; label: string }
    secondary: { settings: string; getHelp: string; search: string }
    user: { account: string; billing: string; notifications: string; language: string; switchLanguage: string; logout: string }
  }
}

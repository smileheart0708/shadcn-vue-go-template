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
    field: { email: string; password: string }
    feedback: { networkError: string; required: string; unknownError: string }
    state: { empty: string; loading: string; noData: string }
  }
  route: { dashboard: string; login: string; notFound: string; settings: string }
  settings: {
    title: string
    description: string
    saved: string
    save: string
    tabs: { account: string; appearance: string; notifications: string }
    account: {
      profile: string
      profileDesc: string
      changeAvatar: string
      avatarHint: string
      name: string
      namePlaceholder: string
      email: string
      emailPlaceholder: string
      edit: string
      editProfile: string
      editProfileDesc: string
      sendCode: string
      verificationCode: string
      verificationCodePlaceholder: string
      password: string
      passwordDesc: string
      currentPassword: string
      currentPasswordPlaceholder: string
      newPassword: string
      newPasswordPlaceholder: string
      confirmPassword: string
      confirmPasswordPlaceholder: string
      updatePassword: string
      privacy: string
      privacyDesc: string
      publicProfile: string
      publicProfileDesc: string
      allowSearch: string
      allowSearchDesc: string
      showActivity: string
      showActivityDesc: string
      dangerZone: string
      dangerZoneDesc: string
      dangerZoneConfirm: string
      deleteAccount: string
      deleteAccountConfirm: string
    }
    appearance: {
      theme: string
      themeDesc: string
      colorTheme: string
      light: string
      dark: string
      system: string
      selectTheme: string
      language: string
      selectLanguage: string
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
      emailPlaceholder: string
      forgotPassword: string
      loginFailed: string
      passwordPlaceholder: string
      rememberMe: string
      signingIn: string
      submit: string
      title: string
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
  sonner: { loginSuccess: string }
  theme: { light: string; dark: string; system: string }
  nav: {
    main: { dashboard: string; lifecycle: string; analytics: string; projects: string; team: string }
    documents: { dataLibrary: string; reports: string; wordAssistant: string; label: string; more: string }
    secondary: { settings: string; getHelp: string; search: string }
    user: { account: string; billing: string; notifications: string; language: string; switchLanguage: string; logout: string }
  }
}

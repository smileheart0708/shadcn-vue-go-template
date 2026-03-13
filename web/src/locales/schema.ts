export interface MessageSchema {
  app: {
    name: string
  }
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
      save: string
      search: string
      submit: string
      update: string
    }
    field: {
      email: string
      password: string
    }
    feedback: {
      networkError: string
      required: string
      unknownError: string
    }
    state: {
      empty: string
      loading: string
      noData: string
    }
  }
  route: {
    dashboard: string
    login: string
    notFound: string
  }
  auth: {
    signIn: {
      description: string
      forgotPassword: string
      rememberMe: string
      submit: string
      title: string
    }
  }
}

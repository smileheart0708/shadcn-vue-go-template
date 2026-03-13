import type { MessageSchema } from '@/locales/schema'

const enUS = {
  app: {
    name: 'shadcn-vue-go-template',
  },
  common: {
    action: {
      add: 'Add',
      back: 'Back',
      cancel: 'Cancel',
      close: 'Close',
      confirm: 'Confirm',
      create: 'Create',
      delete: 'Delete',
      edit: 'Edit',
      save: 'Save',
      search: 'Search',
      submit: 'Submit',
      update: 'Update',
    },
    field: {
      email: 'Email',
      password: 'Password',
    },
    feedback: {
      networkError: 'Network error. Please try again.',
      required: 'This field is required.',
      unknownError: 'Something went wrong. Please try again.',
    },
    state: {
      empty: 'Nothing here yet.',
      loading: 'Loading...',
      noData: 'No data',
    },
  },
  route: {
    dashboard: 'Dashboard',
    login: 'Login',
    notFound: 'Page not found',
  },
  auth: {
    signIn: {
      description: 'Use your account credentials to continue.',
      forgotPassword: 'Forgot password?',
      rememberMe: 'Remember me',
      submit: 'Sign in',
      title: 'Welcome back',
    },
  },
} as const satisfies MessageSchema

export default enUS

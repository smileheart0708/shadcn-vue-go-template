import type { MessageSchema } from '@/locales/schema'

const enUS = {
  app: { name: 'shadcn-vue-go-template' },
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
      menu: 'Open menu',
      save: 'Save',
      search: 'Search',
      submit: 'Submit',
      update: 'Update',
    },
    field: { email: 'Email', password: 'Password' },
    feedback: {
      networkError: 'Network error. Please try again.',
      required: 'This field is required.',
      unknownError: 'Something went wrong. Please try again.',
    },
    state: { empty: 'Nothing here yet.', loading: 'Loading...', noData: 'No data' },
  },
  route: { dashboard: 'Dashboard', login: 'Login', notFound: 'Page not found' },
  auth: {
    signIn: {
      description: 'Use your account credentials to continue.',
      forgotPassword: 'Forgot password?',
      loginFailed: 'Login failed. Please try again.',
      rememberMe: 'Remember me',
      signingIn: 'Signing in...',
      submit: 'Sign in',
      title: 'Welcome back',
    },
  },
  table: {
    action: {
      addSection: 'Add Section',
      customizeColumns: 'Customize Columns',
      columns: 'Columns',
      delete: 'Delete',
      edit: 'Edit',
      favorite: 'Favorite',
      makeCopy: 'Make a copy',
    },
    column: {
      header: 'Header',
      limit: 'Limit',
      reviewer: 'Reviewer',
      sectionType: 'Section Type',
      status: 'Status',
      target: 'Target',
    },
    empty: 'No results.',
    pagination: {
      rowsPerPage: 'Rows per page',
      pageOf: 'Page {page} of {total}',
      goToFirstPage: 'Go to first page',
      goToPreviousPage: 'Go to previous page',
      goToNextPage: 'Go to next page',
      goToLastPage: 'Go to last page',
      rowSelected: '{selected} of {total} row(s) selected.',
    },
    select: { assignReviewer: 'Assign reviewer', view: 'Select a view' },
    tab: {
      outline: 'Outline',
      pastPerformance: 'Past Performance',
      keyPersonnel: 'Key Personnel',
      focusDocuments: 'Focus Documents',
    },
  },
  notFound: { description: 'The page you are looking for does not exist.' },
  sonner: { loginSuccess: 'Login successful!' },
} as const satisfies MessageSchema

export default enUS

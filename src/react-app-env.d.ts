/// <reference types="react-scripts" />

declare namespace NodeJS {
  export interface ProcessEnv {
    REACT_APP_API_URL: string;
    REACT_APP_GA_TRACKING_ID: string;
    REACT_APP_VERSION: string;
    REACT_APP_NAME: string;
  }
}

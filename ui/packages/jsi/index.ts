export interface Post {
  id: number;
  slug: string;
  title: string;
  excerpt: string;
  author: string;
  date: string;
  readTime: number;
  tags: string[];
}

export interface Project {
  id: string;
  title: string;
  description: string;
  tech: string[];
  stars: number;
  status: "active" | "stable" | "beta";
}

export interface Revision {
  rev: string;
  date: string;
  summary: string;
}

export interface Profile {
  id: string;
  name: string;
  email: string;
  role: string;
  bio: string;
  github: string;
  website: string;
}

declare global {
  const __JSI__: {
    getPosts: () => Promise<Post[]>;
    getPost: (slug: string) => Promise<Post>;
    getProjects: () => Promise<Project[]>;
    getProject: (id: string) => Promise<Project>;
    getProfile: (id?: string) => Promise<Profile>;
    getRevisions: (slug: string) => Promise<Revision[]>;
    getRevision: (slug: string, rev: string) => Promise<Revision>;
    triggerError: (message?: string) => Promise<never>;
  };
}

// Per-request context injected by the Go VM before each render.
export declare const __request__: {
  url: string;
  path: string;
  query: string;
  method: string;
  headers: Record<string, string>;
};

// Client manifest injected as a JS define by the bundler.
export declare const __CLIENT_MANIFEST__: Record<string, {
  id: string;
  name: string;
  chunks: string[];
  async: boolean;
}>;

export default __JSI__;

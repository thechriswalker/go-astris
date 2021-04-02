import * as React from "react";
import { DoubleCheck } from "../components/Icons";

export enum Role {
  Authority = "Authority",
  Regsitrar = "Registrar",
  Trustee = "Trustee",
  Voter = "Voter",
  Auditor = "Auditor",
}

const role2class = {
  [Role.Authority]: "is-danger",
  [Role.Regsitrar]: "is-warning",
  [Role.Trustee]: "is-info",
  [Role.Voter]: "is-primary",
  [Role.Auditor]: "is-success",
} as const;

type PageProps = {
  title: string;
  subtitle?: string;
  role: Role;
};

export const Page: React.FC<PageProps> = ({ children, ...props }) => {
  return (
    <div
      className="is-flex is-flex-direction-column"
      style={{ minHeight: "100vh" }}
    >
      <Header {...props} />
      <div className="is-flex-grow-1">{children}</div>
      <Footer />
    </div>
  );
};

export const Header: React.FC<PageProps> = ({ title, subtitle, role }) => {
  return (
    <section className="hero is-dark">
      <div className="hero-body">
        <h1 className="title">
          <span className="icon-text has-text-primary">
            <DoubleCheck />
            <span>Astris </span>
          </span>
          {title}
          <div className="tags has-addons is-pulled-right">
            <span className="tag is-medium">Acting as</span>
            <span className={`tag is-medium ${role2class[role]}`}>{role}</span>
          </div>
        </h1>
        {subtitle && <p className="subtitle">{subtitle}</p>}
      </div>
    </section>
  );
};

export const Footer: React.FC = () => {
  const { version, commit, buildDate } = window.META;
  return (
    <div className="section has-background-light">
      <div className="has-text-centered">
        <p>
          <span className="icon-text">
            <DoubleCheck className="is-small" /> <strong>Astris Voting </strong>
          </span>
          <code>{version}</code>/<code>{commit.slice(0, 8)}</code>/
          <code>{buildDate}</code>
        </p>
        <p>
          <a href="https://opensource.org/licenses/GPL-3.0">GPL-3</a>
          {" — "}
          <a href="https://github.com/thechriswalker/go-astris">
            github.com/thechriswalker/go-astris
          </a>
        </p>
        <p></p>
      </div>
    </div>
  );
};

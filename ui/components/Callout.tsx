import * as React from "react";

export enum CalloutType {
  Info = "info",
  Warning = "warning",
  Danger = "danger",
  Success = "success",
}

type CalloutProps = {
  title: React.ReactNode;
  subtitle?: React.ReactNode;
};

export const Callout: React.FC<{ kind?: CalloutType } & CalloutProps> = ({
  kind = CalloutType.Info,
  title,
  subtitle,
}) => {
  return (
    <section className={`hero is-small is-${kind}`}>
      <div className="hero-body">
        <h1 className="title">{title}</h1>
        {subtitle && <p className="subtitle">{subtitle}</p>}
      </div>
    </section>
  );
};

export const CalloutDanger: React.FC<CalloutProps> = (props) => (
  <Callout kind={CalloutType.Danger} {...props} />
);

export const CalloutInfo: React.FC<CalloutProps> = (props) => (
  <Callout kind={CalloutType.Info} {...props} />
);
export const CalloutWarning: React.FC<CalloutProps> = (props) => (
  <Callout kind={CalloutType.Warning} {...props} />
);
export const CalloutSuccess: React.FC<CalloutProps> = (props) => (
  <Callout kind={CalloutType.Success} {...props} />
);

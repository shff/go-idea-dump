-- Users

create table if not exists users(
  id serial primary key,
  email varchar(255) not null,
  password varchar(255) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(email)
);

create table if not exists sessions(
  id serial primary key,
  user_id integer not null,
  token varchar(255) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (user_id) references users(id)
);

create table if not exists login_providers(
  id serial primary key,
  name varchar(64) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name)
);

create table if not exists logins(
  id serial primary key,
  user_id integer not null,
  provider_id integer not null,
  provider_user varchar(25) not null,
  access_token text not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (user_id) references users(id),
  foreign key (provider_id) references social_login_providers(id)
);

create table if not exists password_reset_tokens(
  id serial primary key,
  user_id integer not null,
  token varchar(25) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (user_id) references users(id)
);

create table if not exists verifications(
  id serial primary key,
  user_id integer not null,
  token varchar(255) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key(user_id) references users(id)
);

create table if not exists accesses(
  id serial primary key,
  user_id integer not null,
  ip_address inet not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (user_id) references users(id)
);

-- Projects

create table if not exists projects(
  id serial primary key,
  name varchar(64) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name)
);

-- External APIs

create table if not exists services(
  id serial primary key,
  project_id integer not null,
  name varchar(64) not null,
  provider varchar(64) not null,
  credentials blob not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, project_id),
  foreign key (project_id) references projects(id)
);

create table if not exists service_log(
  id serial primary key,
  service_id integer not null,
  request blob not null,
  response blob not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (service_id) references services(id)
);

-- Organizations

create table if not exists organizations(
  id serial primary key,
  project_id integer not null,
  name varchar(64) not null,
  main integer not null default false,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, project_id),
  foreign key (project_id) references projects(id)
);

create table if not exists memberships(
  id serial primary key,
  organization_id integer not null,
  user_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(organization_id, user_id),
  foreign key (organization_id) references organizations(id),
  foreign key (user_id) references users(id)
);

-- Resources

create table if not exists resource_type(
  id serial primary key,
  project_id integer not null,
  name varchar(64) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, project_id),
  foreign key (project_id) references projects(id)
);

create table if not exists actions(
  id serial primary key,
  resource_type_id integer not null,
  name varchar(64) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, resource_type_id),
  foreign key (resource_type_id) references resource_type(id)
);

create table if not exists resources(
  id serial primary key,
  resource_type_id integer not null,
  resource_key varchar(64) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(resource_key, resource_type_id),
  foreign key (resource_type_id) references resource_type(id)
);

create table if not exists log(
  id serial primary key,
  resource_id integer not null,
  action_id integer not null,
  user_id integer not null,
  data blob not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (resource_id) references resources(id),
  foreign key (action_id) references actions(id),
  foreign key (user_id) references users(id)
);

-- Permissions

create table if not exists roles(
  id serial primary key,
  project_id integer not null,
  name varchar(64) not null,
  description text not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(project_id, name),
  foreign key (project_id) references projects(id)
);

create table if not exists permissions(
  id serial primary key,
  project_id integer not null,
  name varchar(64) not null,
  description text not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(project_id, name),
  foreign key (project_id) references projects(id)
);

create table if not exists role_permissions(
  id serial primary key,
  role_id integer not null,
  permission_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(role_id, permission_id),
  foreign key (role_id) references roles(id),
  foreign key (permission_id) references permissions(id)
);

create table if not exists scopes(
  id serial primary key,
  resource_type_id integer not null,
  name varchar(64) not null,
  description text not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, resource_type_id),
  foreign key (resource_type_id) references resource_type(id)
);

create table if not exists policies(
  id serial primary key,
  permission_id integer not null,
  action_id integer not null,
  scope_id integer,
  condition blob not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(permission_id, action_id, scope_id),
  foreign key (permission_id) references permissions(id),
  foreign key (action_id) references actions(id),
  foreign key (scope_id) references scopes(id)
);

create table if not exists user_roles(
  id serial primary key,
  role_id integer not null,
  user_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(role_id, user_id),
  foreign key (role_id) references roles(id),
  foreign key (user_id) references users(id)
);

create table if not exists user_permissions(
  id serial primary key,
  permission_id integer not null,
  user_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(permission_id, user_id),
  foreign key (permission_id) references permissions(id),
  foreign key (user_id) references users(id)
);

-- Content

create table if not exists channels(
  id serial primary key,
  project_id integer not null,
  service_id integer not null,
  name varchar(64) not null,
  kind varchar(32) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, project_id),
  foreign key (project_id) references projects(id),
  foreign key (service_id) references services(id)
);

create table if not exists languages(
  id serial primary key,
  project_id integer not null,
  name varchar(64) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, project_id),
  foreign key (project_id) references projects(id)
);

create table if not exists content_types(
  id serial primary key,
  project_id integer not null,
  default_language_id integer not null,
  name varchar(64) not null,
  kind varchar(32) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, project_id),
  foreign key (project_id) references projects(id),
  foreign key (default_language_id) references languages(id)
);

create table if not exists contents(
  id serial primary key,
  content_type_id integer not null,
  resource_type_id integer,
  name varchar(64) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, content_type_id),
  foreign key (content_type_id) references content_types(id)
);

create table if not exists translations(
  id serial primary key,
  content_id integer not null,
  channel_id integer not null,
  language_id integer not null,
  content text not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(content_id, channel_id, language_id),
  foreign key (content_id) references contents(id),
  foreign key (channel_id) references channels(id),
  foreign key (language_id) references languages(id)
);

-- Notifications

create table if not exists notification_types(
  id serial primary key,
  project_id integer not null,
  content_id integer not null,
  name varchar(64) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  unique(name, project_id),
  foreign key (project_id) references projects(id),
  foreign key (content_id) references content(id)
);

create table if not exists notification_settings(
  id serial primary key,
  user_id integer not null,
  notification_type_id integer not null,
  enabled integer not null default true,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(user_id, notification_type_id),
  foreign key (user_id) references users(id),
  foreign key (notification_type_id) references notification_types(id)
);

create table if not exists notifications(
  id serial primary key,
  user_id integer not null,
  notification_type_id integer not null,
  resource_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  read_at timestamp,
  foreign key (user_id) references users(id),
  foreign key (notification_type_id) references notification_types(id),
  foreign key (resource_id) references resources(id)
);

-- Resource Enrichment

create table if not exists attachments(
  id serial primary key,
  resource_id integer not null,
  service_id integer not null,
  name varchar(128) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, resource_id),
  unique(name, service_id),
  foreign key (resource_id) references resources(id),
  foreign key (service_id) references services(id)
);

create table if not exists watches(
  id serial primary key,
  resource_id integer not null,
  user_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (resource_id) references resources(id),
  foreign key (user_id) references users(id)
);

create table if not exists tags(
  id serial primary key,
  resource_type_id integer not null,
  scope_id integer not null,
  name varchar(64) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, resource_type_id),
  foreign key (resource_type_id) references projects(id)
);

create table if not exists resource_tags(
  id serial primary key,
  tag_id integer not null,
  resource_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(tag_id, resource_id),
  foreign key (tag_id) references tags(id),
  foreign key (resource_id) references resources(id)
);

create table if not exists comments(
  id serial primary key,
  resource_id integer not null,
  user_id integer not null,
  parent_id integer,
  scope_id integer not null,
  comment text not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (resource_id) references resources(id),
  foreign key (user_id) references users(id),
  foreign key (parent_id) references comments(id)
);

create table if not exists notes(
  id serial primary key,
  resource_id integer not null,
  user_id integer not null,
  scope_id integer not null,
  description text not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (resource_id) references resources(id),
  foreign key (user_id) references users(id)
);

create table if not exists tasks(
  id serial primary key,
  resource_id integer not null,
  user_id integer not null,
  assignee_id integer not null,
  scope_id integer not null,
  description text not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  due_at timestamp not null,
  completed_at timestamp,
  foreign key (resource_id) references resources(id),
  foreign key (user_id) references users(id),
  foreign key (assignee_id) references users(id)
);

-- CRM

create table if not exists persons(
  id serial primary key,
  project_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(project_id),
  foreign key (project_id) references projects(id)
);

create table if not exists person_fields(
  id serial primary key,
  project_id integer not null,
  name varchar(64) not null,
  kind varchar(32) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, project_id),
  foreign key (project_id) references projects(id)
);

create table if not exists person_information(
  id serial primary key,
  person_id integer not null,
  field_id integer not null,
  value text not null,
  main integer not null default false,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  invalidated_at timestamp,
  unique(person_id, field_id),
  foreign key (person_id) references persons(id),
  foreign key (field_id) references fields(id)
);

-- Messaging

create table if not exists conversations(
  id serial primary key,
  project_id integer not null,
  channel_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (project_id) references projects(id),
  foreign key (channel_id) references channels(id)
);

create table if not exists participants(
  id serial primary key,
  conversation_id integer not null,
  user_id integer,
  person_id integer,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(conversation_id, user_id, person_id),
  foreign key (conversation_id) references conversations(id),
  foreign key (user_id) references users(id),
  foreign key (person_id) references persons(id),
  check ((user_id is not null and person_id is null) or (user_id is null and person_id is not null))
);

create table if not exists messages(
  id serial primary key,
  conversation_id integer not null,
  participant_id integer not null,
  content text not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (conversation_id) references conversations(id),
  foreign key (participant_id) references participants(id)
);

-- Workflows

create table if not exists workflows(
  id serial primary key,
  project_id integer not null,
  name varchar(64) not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, project_id),
  foreign key (project_id) references projects(id)
);

create table if not exists workflow_steps(
  id serial primary key,
  workflow_id integer not null,
  name varchar(64) not null,
  content blob not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(name, workflow_id),
  foreign key (workflow_id) references workflows(id)
);

create table if not exists workflow_transitions(
  id serial primary key,
  workflow_id integer not null,
  from_step_id integer not null,
  to_step_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  unique(from_step_id, to_step_id),
  foreign key (workflow_id) references workflows(id),
  foreign key (from_step_id) references workflow_steps(id),
  foreign key (to_step_id) references workflow_steps(id)
);

create table if not exists workflow_instances(
  id serial primary key,
  workflow_id integer not null,
  resource_id integer not null,
  current_step_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (workflow_id) references workflows(id),
  foreign key (resource_id) references resources(id),
  foreign key (current_step_id) references workflow_steps(id)
);

create table if not exists workflow_history(
  id serial primary key,
  workflow_instance_id integer not null,
  transition_id integer not null,
  created_at timestamp not null default current_timestamp,
  updated_at timestamp not null default current_timestamp,
  deleted_at timestamp,
  foreign key (workflow_instance_id) references workflow_instances(id),
  foreign key (transition_id) references workflow_transitions(id)
);

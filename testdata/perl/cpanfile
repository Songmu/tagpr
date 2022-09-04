requires 'App::Wallflower', '1.005';
requires 'File::Copy::Recursive';
requires 'Git::Repository';
requires 'Git::Repository::FileHistory', '0.03';
requires 'HTTP::Date';
requires 'IPC::Cmd';
requires 'List::UtilsBy';
requires 'MIME::Base32', '1.301';
requires 'Module::Functions';
requires 'Mouse';
requires 'Object::Container';
requires 'Path::Tiny', '0.061';
requires 'Plack';
requires 'Puncheur', 'v0.3.0';
requires 'Router::Boom', '1.00';
requires 'String::CamelCase';
requires 'Text::Markdown::Discount', '0.10';
requires 'Text::Markup::Any';
requires 'Text::Xslate';
requires 'Time::Piece';
requires 'URI';
requires 'URI::tag';
requires 'XML::FeedPP';
requires 'YAML::Tiny';

on configure => sub {
    requires 'perl', '5.010';
    requires 'Module::Build::Tiny', '0.035';
};

on test => sub {
    requires 'File::pushd';
    requires 'Scope::Guard';
    requires 'Test::Mock::Guard';
    requires 'Test::More', '0.98';
    requires 'Test::Output';

    recommends 'Capture::Tiny';
    recommends 'Class::Unload';
};

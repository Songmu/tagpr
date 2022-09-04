package Riji;
use 5.010;
use strict;
use warnings;
use Puncheur::Lite;

use Encode;
use File::Spec;
use YAML::Tiny ();

use version 0.77; our $VERSION = version->declare("v1.0.0");

__PACKAGE__->setting(
    handle_static => 1,
);
__PACKAGE__->load_plugins(qw/Model ShareDir/);

sub base_dir { state $b = File::Spec->rel2abs('./') }

sub load_config {
    my $self = shift;
    my $file = File::Spec->catfile($self->base_dir, 'riji.yml');
    unless (-e $file) {
        die sprintf "config file: [%s] not found.\n", $file;
    }
    YAML::Tiny::LoadFile($file);
}

get '/{match:(?:[-_a-zA-Z0-9]+(?:\.[0-9]+)?.html)?}' => sub {
    my ($c, $args) = @_;

    my $match = $args->{match} || 'index.html';
    my ($basename, $page) = $match =~ m!^([-_a-zA-Z0-9]+)(?:\.([0-9]+))?\.html$!;

    my $blog    = $c->model('Blog');
    my $article = $blog->article($basename, {$page ? (page => $page) : ()});

    if (!$article && $basename ne 'index') {
        return $c->res_404;
    }

    my $tmpl = $article && $article->template;
    unless (defined $tmpl) {
        $tmpl   = $basename if $basename eq 'index';
        $tmpl //= 'default';
    }
    $tmpl .= '.tx' unless $tmpl =~ /\.tx$/;

    $c->render($tmpl, {
        blog    => $blog,
        page    => $page,
        article => $article,
    });
};

my $s = '[-_a-zA-Z0-9]+';
get "/entry/{name:$s(?:\.$s)*(?:/$s(?:\.$s)*)*}.html" => sub {
    my ($c, $args) = @_;

    my $name = $args->{name};
    my $blog = $c->model('Blog');
    my $entry = $blog->entry($name);
    return $c->res_404 unless $entry;

    my $tmpl = $entry->template // 'entry';
    $tmpl .= '.tx' unless $tmpl =~ /\.tx$/;

    $c->render($tmpl, {
        blog    => $blog,
        entry   => $entry,
    });
};

get '/tag/:tag.html' => sub {
    my ($c, $args) = @_;

    my $tag = $args->{tag};
    my $blog = $c->model('Blog');
    $tag = $blog->tag($tag);
    return $c->res_404 unless $tag;

    $c->render('tag.tx', {
        blog  => $blog,
        tag   => $tag,
    });
};

get '/atom.xml' => sub {
    my $c = shift;

    my $atom = $c->model('Blog')->atom;
    my $xml = $atom->feed->to_string;
    my $atom_url = $atom->site_url . '/atom.xml';
    $atom_url =~ s{//atom\.xml$}{/atom.xml};
    $xml =~ s{(\s+)(<link[^>]+>)}{$1$2$1<link rel="self" type="application/atom+xml" href="$atom_url" />};
    $c->create_response(200, ['Content-Type' => 'application/atom+xml'], [encode($c->encoding, $xml)]);
};

sub get_functions {
    my $self = shift;

    state %functions;
    my $functionspl = File::Spec->catfile($self->base_dir, 'share', 'functions.pl');
    if (-f -r $functionspl && !%functions) {
        my $code = do {
            local $/;
            open my $fh, '<', $functionspl or die $!;
            <$fh>
        };
        my $package = 'Riji::_Sandbox::Functions';
        eval <<"..."; ## no critic
        package $package;
        use strict;
        use warnings;
        use utf8;

        $code
        1;
...
        if (my $err = $@) {
            die "$err\n";
        }
        require Module::Functions;
        my @functions = Module::Functions::get_public_functions($package);
        for my $func (@functions) {
            $functions{$func} = $package->can($func);
        }
    }
    %functions;
}

sub create_view {
    my $self = shift;

    Text::Xslate->new(
        path => $self->template_dir,
        module   => [
            'Text::Xslate::Bridge::Star',
        ],
        function => {
            c         => sub { $self->context },
            uri_for   => sub { $self->context->uri_for(@_) },
            uri_with  => sub { $self->context->req->uri_with(@_) },
            $self->get_functions,
        },
        ($self->debug_mode ? ( warn_handler => sub {
            Text::Xslate->print( # print method escape html automatically
                '[[', @_, ']]',
            );
        } ) : () ),
    );
}

1;
__END__

=for stopwords cli riji

=encoding utf-8

=head1 NAME

Riji - Simple, git based blog tool

=head1 SYNOPSIS

    % cpanm -qn Riji           # install `riji` cli
    % rjji setup               # setup new blog site
    % $EDITOR riji.yml         # adjust configuration
    % riji new-entry your-slug # create new blog entry in Markdown
    % git add article/ && git commit -m "add new entry"
    % riji server              # local server for staging starts on the port 3650.
    % riji publish             # static site will be created in the ./riji directory

=head1 TUTORIAL

Japanese: L<http://songmu.github.io/p5-Riji/blog/>

English L<http://perlmaven.com/blogging-with-riji>

=head1 DESCRIPTION

Riji is a static site generator using Markdown, featuring RSS generation from git history.

'Riji'(日记) means diary in Chinese.

=head1 FEATURES

=over

=item Static site generation with Markdown files.

=item All operations can be performed with the cli "riji".

=item Commits Markdown files to your git repository and automatically generates RSS from the git log.

=item Name of markdown file will be directly mapped to the URL as HTML.

=item YAML Frontmatter can be written optionally in Markdown file for meta-information, like tags, etc.

=item Customizable site template with Text::Xslate Kolon format.

=item Kolon template notation can also be used in Markdown files.

=item Your own template macros can be defined in the functions.pl file.

=back

=head1 DOCKER

docker container is also available.

    % docker run --rm -v $(PWD):/riji -v $(PWD)/.git:/riji/.git -i ghcr.io/songmu/riji publish

=head1 LICENSE

Copyright (C) Masayuki Matsuki.

This library is free software; you can redistribute it and/or modify
it under the same terms as Perl itself.

=head1 THANKS

Thanks to Gabor Szabo E<lt>szabgab@gmail.comE<gt> for great English tutorial.

=head1 AUTHOR

Masayuki Matsuki E<lt>y.songmu@gmail.comE<gt>

=cut

